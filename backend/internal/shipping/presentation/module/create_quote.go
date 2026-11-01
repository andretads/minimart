package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// CreateQuote creates a new shipping quote. It is idempotent: a repeated call
// with the same idempotency key and parameters returns the quote that already
// exists.
func (s *Shipping) CreateQuote(ctx context.Context, req shippingclient.CreateQuoteRequest) (shippingclient.ShippingQuote, error) {
	q, err := s.svc.CreateQuote(ctx, application.CreateQuoteRequest{
		CustomerID:     req.CustomerID,
		MethodID:       req.MethodID,
		AddressID:      req.AddressID,
		SubtotalCents:  req.SubtotalCents,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteNotFound
		}
		if errors.Is(err, domain.ErrAddressNotFound) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteNotFound // Map to quote not found
		}
		if errors.Is(err, domain.ErrMethodNotFound) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrMethodNotFound
		}
		if errors.Is(err, domain.ErrMethodInactive) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrMethodInactive
		}
		if errors.Is(err, domain.ErrZoneNotFound) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrZoneNotFound
		}
		if errors.Is(err, domain.ErrZoneInactive) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrZoneInactive
		}
		if errors.Is(err, domain.ErrRateNotFound) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrRateNotFound
		}
		if errors.Is(err, domain.ErrRateInactive) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrRateInactive
		}
		if errors.Is(err, domain.ErrNoMatchingRate) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrNoMatchingRate
		}
		if errors.Is(err, domain.ErrIdempotencyConflict) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrIdempotencyConflict
		}
		return shippingclient.ShippingQuote{}, err
	}
	return toClientShippingQuote(q), nil
}
