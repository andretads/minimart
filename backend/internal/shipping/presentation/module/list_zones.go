package module

import (
	"context"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// ListZones returns all shipping zones.
func (s *Shipping) ListZones(ctx context.Context) ([]shippingclient.ShippingZone, error) {
	zones, err := s.svc.ListZones(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]shippingclient.ShippingZone, len(zones))
	for i, z := range zones {
		resp[i] = toClientShippingZone(z)
	}
	return resp, nil
}
