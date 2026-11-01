package module

import (
	"context"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// ListMethods returns all shipping methods.
func (s *Shipping) ListMethods(ctx context.Context) ([]shippingclient.ShippingMethod, error) {
	methods, err := s.svc.ListMethods(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]shippingclient.ShippingMethod, len(methods))
	for i, m := range methods {
		resp[i] = toClientShippingMethod(m)
	}
	return resp, nil
}
