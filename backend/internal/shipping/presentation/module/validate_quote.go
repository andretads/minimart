package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// ValidateQuote validates a shipping quote for a customer.
func (s *Shipping) ValidateQuote(ctx context.Context, req shippingclient.ValidateQuoteRequest) (shippingclient.ShippingQuote, error) {
	q, err := s.svc.ValidateQuote(ctx, req.ID, req.CustomerID)
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
		return shippingclient.ShippingQuote{}, err
	}
	return toClientShippingQuote(q), nil
}
