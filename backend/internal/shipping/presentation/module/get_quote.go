package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// GetQuote returns a shipping quote by ID.
func (s *Shipping) GetQuote(ctx context.Context, req shippingclient.GetQuoteRequest) (shippingclient.ShippingQuote, error) {
	q, err := s.svc.GetQuote(ctx, req.ID)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			return shippingclient.ShippingQuote{}, shippingclient.ErrQuoteNotFound
		}
		return shippingclient.ShippingQuote{}, err
	}
	return toClientShippingQuote(q), nil
}
