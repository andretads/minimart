package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// PriceQuote prices an existing quote for the subtotal the caller holds.
func (s *Shipping) PriceQuote(ctx context.Context, req shippingclient.PriceQuoteRequest) (shippingclient.ShippingQuote, error) {
	q, err := s.svc.PriceQuote(ctx, req.ID, req.CustomerID, req.SubtotalCents)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteNotFound
		}
		if errors.Is(err, domain.ErrQuoteExpired) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteExpired
		}
		if errors.Is(err, domain.ErrQuoteConsumed) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteConsumed
		}
		if errors.Is(err, domain.ErrInvalidQuoteDestination) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrInvalidQuote
		}
		if errors.Is(err, domain.ErrOwnershipMismatch) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrInvalidQuote
		}
		if errors.Is(err, domain.ErrRateInactive) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrRateInactive
		}
		if errors.Is(err, domain.ErrNoMatchingRate) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrNoMatchingRate
		}
		return shippingclient.ShippingQuote{}, err
	}
	return toClientShippingQuote(q), nil
}
