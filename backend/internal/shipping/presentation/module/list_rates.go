package module

import (
	"context"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// ListRates returns all shipping rates for a zone and method.
func (s *Shipping) ListRates(ctx context.Context, zoneID, methodID string) ([]shippingclient.ShippingRate, error) {
	rates, err := s.svc.ListRates(ctx, zoneID, methodID)
	if err != nil {
		return nil, err
	}
	resp := make([]shippingclient.ShippingRate, len(rates))
	for i, r := range rates {
		resp[i] = toClientShippingRate(r)
	}
	return resp, nil
}
