package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// GetRate returns a shipping rate by ID.
func (s *Shipping) GetRate(ctx context.Context, id string) (shippingclient.ShippingRate, error) {
	r, err := s.svc.GetRate(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrRateNotFound) {
			return shippingclient.ShippingRate{}, shippingclient.ErrRateNotFound
		}
		return shippingclient.ShippingRate{}, err
	}
	return toClientShippingRate(r), nil
}
