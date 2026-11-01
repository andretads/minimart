package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// GetZone returns a shipping zone by ID.
func (s *Shipping) GetZone(ctx context.Context, id string) (shippingclient.ShippingZone, error) {
	z, err := s.svc.GetZone(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrZoneNotFound) {
			return shippingclient.ShippingZone{}, shippingclient.ErrZoneNotFound
		}
		return shippingclient.ShippingZone{}, err
	}
	return toClientShippingZone(z), nil
}
