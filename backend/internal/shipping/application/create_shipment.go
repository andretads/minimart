package application

import (
	"context"
	"errors"
	"minimart/internal/shipping/domain"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"strconv"
)

type CreateShipmentRequest struct {
	OrderID             string
	UserID              string
	MethodID            string
	DestinationSnapshot domain.DestinationSnapshot
	ShippingCostCents   int64
}

func (s *Service) CreateShipment(ctx context.Context, req CreateShipmentRequest) (domain.Shipment, error) {
	shipment, err := domain.NewShipment(req.OrderID, req.UserID, req.MethodID, req.DestinationSnapshot, req.ShippingCostCents)
	if err != nil {
		s.m.IncShipmentCreate(metrics.StatusError)
		return domain.Shipment{}, err
	}

	// Try to create - relies on unique constraint on order_id
	if err := s.repo.CreateShipment(ctx, shipment); err != nil {
		s.m.IncShipmentCreate(metrics.StatusError)
		if errors.Is(err, domain.ErrShipmentExists) {
			return domain.Shipment{}, domain.ErrShipmentExists
		}
		return domain.Shipment{}, err
	}

	s.m.IncShipmentCreate(metrics.StatusSuccess)
	return shipment, nil
}

// CreateShipmentFromOrderPaid creates the shipment for a paid order.
//
// It is idempotent: a redelivered order-paid event replays the shipment that
// already exists instead of creating a second one. Redis is a fast cache in
// front of the database; if it is unavailable the flow falls back to the
// shipments table's unique index on order_id, which is the authoritative guard.
func (s *Service) CreateShipmentFromOrderPaid(ctx context.Context, orderID, userID, customerID, quoteID, methodID string, destination domain.DestinationSnapshot, totalCents int64) (domain.Shipment, error) {
	key := shipmentKey(orderID)
	fp := shipmentFingerprint(orderID, customerID, quoteID, methodID, totalCents, destination)

	// Fast path: has this order already produced a shipment?
	existing, done, err := s.existingShipment(ctx, key, fp, orderID)
	if err != nil {
		s.m.IncShipmentCreateFromOrderPaid(metrics.StatusError)
		return domain.Shipment{}, err
	}
	if done {
		s.m.IncShipmentCreateFromOrderPaid(metrics.StatusSuccess)
		return existing, nil
	}

	// Claim the key so a redelivered event fails fast instead of creating a
	// second shipment. On a store error this fails open.
	if !s.shipmentCoord.Reserve(ctx, key, fp) {
		existing, done, lookupErr := s.existingShipment(ctx, key, fp, orderID)
		if lookupErr != nil {
			s.m.IncShipmentCreateFromOrderPaid(metrics.StatusError)
			return domain.Shipment{}, lookupErr
		}
		if done {
			s.m.IncShipmentCreateFromOrderPaid(metrics.StatusSuccess)
			return existing, nil
		}
		s.m.IncShipmentCreateFromOrderPaid(metrics.StatusError)
		return domain.Shipment{}, domain.ErrShipmentInProgress
	}

	shipment, err := s.createShipmentFromOrderPaid(ctx, orderID, userID, customerID, quoteID, methodID, destination, fp)
	if err != nil {
		// A concurrent duplicate won the race and created the shipment. Look it
		// up in the database, not the cache.
		if errors.Is(err, domain.ErrShipmentExists) {
			existing, done, lookupErr := s.lookupShipment(ctx, orderID, fp)
			if lookupErr != nil {
				s.m.IncShipmentCreateFromOrderPaid(metrics.StatusError)
				return domain.Shipment{}, lookupErr
			}
			if done {
				s.shipmentCoord.Complete(ctx, key, fp, existing.ID)
				s.m.IncShipmentCreateFromOrderPaid(metrics.StatusSuccess)
				return existing, nil
			}
		}
		// Free the reservation so a later redelivery may proceed.
		s.shipmentCoord.Release(ctx, key, fp)
		s.m.IncShipmentCreateFromOrderPaid(metrics.StatusError)
		return domain.Shipment{}, err
	}

	s.shipmentCoord.Complete(ctx, key, fp, shipment.ID)
	s.m.IncShipmentCreateFromOrderPaid(metrics.StatusSuccess)
	return shipment, nil
}

// createShipmentFromOrderPaid validates the quote and the method, then creates
// the shipment, consumes the quote and records the shipment-created event in
// one transaction.
func (s *Service) createShipmentFromOrderPaid(ctx context.Context, orderID, userID, customerID, quoteID, methodID string, destination domain.DestinationSnapshot, fp string) (domain.Shipment, error) {
	// Validate quote
	quote, err := s.repo.GetQuote(ctx, quoteID)
	if err != nil {
		return domain.Shipment{}, domain.ErrQuoteNotFound
	}
	if quote.CustomerID != customerID {
		return domain.Shipment{}, domain.ErrOwnershipMismatch
	}
	if quote.Status != domain.QuoteStatusActive {
		if quote.Status == domain.QuoteStatusExpired {
			return domain.Shipment{}, domain.ErrQuoteExpired
		}
		return domain.Shipment{}, domain.ErrQuoteConsumed
	}

	// Validate method
	method, err := s.repo.GetMethod(ctx, methodID)
	if err != nil {
		return domain.Shipment{}, domain.ErrMethodNotFound
	}
	if !method.Active {
		return domain.Shipment{}, domain.ErrMethodInactive
	}

	// Create shipment with the validated quote's cost
	shipment, err := domain.NewShipment(orderID, userID, methodID, quote.DestinationSnapshot, quote.ShippingCostCents)
	if err != nil {
		return domain.Shipment{}, err
	}

	if err := quote.Consume(); err != nil {
		return domain.Shipment{}, err
	}
	if err := s.repo.CreateShipmentForQuote(ctx, shipment, quote); err != nil {
		return domain.Shipment{}, err
	}

	return shipment, nil
}

// existingShipment resolves an idempotency key to the shipment it already
// produced. A store error fails open to the database lookup.
func (s *Service) existingShipment(ctx context.Context, key, fp, orderID string) (domain.Shipment, bool, error) {
	return s.shipmentCoord.Existing(ctx, key, fp)
}

// lookupShipment is the authoritative fallback: it consults the shipments
// table directly, so idempotency holds even when Redis is empty or
// unavailable.
func (s *Service) lookupShipment(ctx context.Context, orderID, fp string) (domain.Shipment, bool, error) {
	shipment, err := s.repo.GetShipmentByOrderID(ctx, orderID)
	if err != nil {
		if errors.Is(err, domain.ErrShipmentNotFound) {
			return domain.Shipment{}, false, nil
		}
		return domain.Shipment{}, false, err
	}
	return shipment, true, nil
}

// reserve claims the idempotency key. It fails open: on a store error it
// returns true so processing continues, relying on the database's unique index.
// shipmentKey is the idempotency key under which the shipment created for an
// order is remembered.
func shipmentKey(orderID string) string {
	return "shipping:shipment:" + orderID
}

// shipmentFingerprint derives the idempotency fingerprint from the fields that
// define the order-paid payload. Reusing a key with any of these changed is a
// conflict.
func shipmentFingerprint(orderID, customerID, quoteID, methodID string, totalCents int64, destination domain.DestinationSnapshot) string {
	return idempotency.Fingerprint(
		orderID,
		customerID,
		quoteID,
		methodID,
		strconv.FormatInt(totalCents, 10),
		destination.Label,
		destination.FullName,
		destination.Street,
		destination.City,
		destination.State,
		destination.Zip,
		destination.Country,
		destination.Phone,
	)
}
