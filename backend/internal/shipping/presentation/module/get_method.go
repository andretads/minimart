package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// GetMethod returns a shipping method by ID.
func (s *Shipping) GetMethod(ctx context.Context, id string) (shippingclient.ShippingMethod, error) {
	m, err := s.svc.GetMethod(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrMethodNotFound) {
			return shippingclient.ShippingMethod{}, shippingclient.ErrMethodNotFound
		}
		return shippingclient.ShippingMethod{}, err
	}
	return toClientShippingMethod(m), nil
}
