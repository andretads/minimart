package application

import (
	"context"
	"errors"
	"minimart/internal/shipping/domain"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"strconv"

	addressclient "minimart/internal/address/presentation/module/client"
)

type CreateQuoteRequest struct {
	CustomerID     string
	MethodID       string
	AddressID      string
	SubtotalCents  int64
	IdempotencyKey string
}

// CreateQuote prices a quote for a customer's address and method.
//
// It is idempotent: a repeated request with the same key and parameters replays
// the quote that already exists; the same key with different parameters is a
// conflict. Redis is a fast cache in front of the database; if it is unavailable
// the flow falls back to the shipping_quotes table's unique index on
// idempotency_key, which is the authoritative guard.
func (s *Service) CreateQuote(ctx context.Context, req CreateQuoteRequest) (domain.ShippingQuote, error) {
	quote, err := s.createQuote(ctx, req)
	if err != nil {
		s.m.IncShippingQuoteCreate(metrics.StatusError)
		return domain.ShippingQuote{}, err
	}
	s.m.IncShippingQuoteCreate(metrics.StatusSuccess)
	return quote, nil
}

func (s *Service) createQuote(ctx context.Context, req CreateQuoteRequest) (domain.ShippingQuote, error) {
	if req.IdempotencyKey == "" {
		return domain.ShippingQuote{}, domain.ErrIdempotencyKeyRequired
	}
	key := quoteKey(req.IdempotencyKey)
	fp := quoteFingerprint(req)

	// Fast path: has this key already produced a quote?
	quote, done, err := s.existingQuote(ctx, key, fp)
	if err != nil {
		return domain.ShippingQuote{}, err
	}
	if done {
		return quote, nil
	}

	// Claim the key so a concurrent duplicate fails fast instead of creating a
	// second quote. On a store error this fails open.
	if !s.quoteCoord.Reserve(ctx, key, fp) {
		quote, done, err = s.existingQuote(ctx, key, fp)
		if err != nil {
			return domain.ShippingQuote{}, err
		}
		if done {
			return quote, nil
		}
		return domain.ShippingQuote{}, domain.ErrQuoteInProgress
	}

	quote, err = s.processQuote(ctx, req, key)
	if err != nil {
		// A concurrent duplicate won the race and created the quote. Look it up
		// in the database, not the cache.
		if errors.Is(err, domain.ErrIdempotencyConflict) {
			existing, done, lookupErr := s.lookupQuoteByKey(ctx, key, fp)
			if lookupErr != nil {
				return domain.ShippingQuote{}, lookupErr
			}
			if done {
				s.quoteCoord.Complete(ctx, key, fp, existing.ID)
				return existing, nil
			}
		}
		// Free the reservation so a later retry may proceed.
		s.quoteCoord.Release(ctx, key, fp)
		return domain.ShippingQuote{}, err
	}

	s.quoteCoord.Complete(ctx, key, fp, quote.ID)
	return quote, nil
}

// processQuote validates the method and the address, prices the quote and
// stores it under the request's idempotency key.
func (s *Service) processQuote(ctx context.Context, req CreateQuoteRequest, key string) (domain.ShippingQuote, error) {
	if req.CustomerID == "" || req.MethodID == "" || req.AddressID == "" {
		return domain.ShippingQuote{}, domain.ErrInvalidQuote
	}
	if req.SubtotalCents < 0 {
		return domain.ShippingQuote{}, domain.ErrInvalidQuote
	}

	// Validate method exists and is active
	method, err := s.repo.GetMethod(ctx, req.MethodID)
	if err != nil {
		return domain.ShippingQuote{}, domain.ErrMethodNotFound
	}
	if !method.Active {
		return domain.ShippingQuote{}, domain.ErrMethodInactive
	}

	// Use address contract to get the address and verify ownership
	addr, err := s.modules.GetAddress(ctx, addressclient.GetAddressRequest{ID: req.AddressID, CustomerID: req.CustomerID})
	if err != nil {
		return domain.ShippingQuote{}, domain.ErrAddressNotFound
	}

	// Build destination snapshot
	dest := domain.DestinationSnapshot{
		Label:    addr.Label,
		FullName: addr.FullName,
		Street:   addr.Street,
		City:     addr.City,
		State:    addr.State,
		Zip:      addr.Zip,
		Country:  addr.Country,
		Phone:    addr.Phone,
	}

	rate, err := s.resolveRate(ctx, dest, req.MethodID)
	if err != nil {
		return domain.ShippingQuote{}, err
	}

	// The subtotal is what the customer is shopping with; the rate decides
	// whether that subtotal unlocks free shipping.
	shippingCost := rate.CalculateCost(req.SubtotalCents)

	quote, err := domain.NewShippingQuote(req.CustomerID, req.MethodID, req.AddressID, key, dest, shippingCost, req.SubtotalCents, rate.FreeThresholdCents, method.EstimatedMinDays, method.EstimatedMaxDays)
	if err != nil {
		return domain.ShippingQuote{}, err
	}

	if err := s.repo.CreateQuote(ctx, quote); err != nil {
		return domain.ShippingQuote{}, err
	}

	return quote, nil
}

// existingQuote resolves an idempotency key to the quote it already produced.
// A store error fails open to the database lookup.
func (s *Service) existingQuote(ctx context.Context, key, fp string) (domain.ShippingQuote, bool, error) {
	return s.quoteCoord.Existing(ctx, key, fp)
}

// lookupQuoteByKey is the authoritative fallback: it consults the
// shipping_quotes table directly, so idempotency holds even when Redis is
// empty or unavailable.
func (s *Service) lookupQuoteByKey(ctx context.Context, key, fp string) (domain.ShippingQuote, bool, error) {
	quote, err := s.repo.GetQuoteByIdempotencyKey(ctx, key)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			return domain.ShippingQuote{}, false, nil
		}
		return domain.ShippingQuote{}, false, err
	}
	if quoteFingerprintOf(quote.CustomerID, quote.MethodID, quote.AddressID, quote.SubtotalCents) != fp {
		return domain.ShippingQuote{}, false, domain.ErrIdempotencyConflict
	}
	return quote, true, nil
}

// quoteKey is the idempotency key under which the quote created for a request
// is remembered. It is namespaced so a key used for another domain's request
// cannot collide with a shipping quote.
func quoteKey(clientKey string) string {
	return "shipping:quote:" + clientKey
}

// quoteFingerprint derives the idempotency fingerprint from the fields that
// define a quote request. Reusing a key with any of these changed is a
// conflict.
func quoteFingerprint(req CreateQuoteRequest) string {
	return quoteFingerprintOf(req.CustomerID, req.MethodID, req.AddressID, req.SubtotalCents)
}

// quoteFingerprintOf fingerprints a stored quote so it can be compared with an
// incoming request.
func quoteFingerprintOf(customerID, methodID, addressID string, subtotalCents int64) string {
	return idempotency.Fingerprint(
		customerID,
		methodID,
		addressID,
		strconv.FormatInt(subtotalCents, 10),
	)
}
