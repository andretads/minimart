package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// GetShipment returns a shipment by ID.
func (s *Shipping) GetShipment(ctx context.Context, id string) (shippingclient.Shipment, error) {
	sh, err := s.svc.GetShipment(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrShipmentNotFound) {
			return shippingclient.Shipment{}, shippingclient.ErrShipmentNotFound
		}
		return shippingclient.Shipment{}, err
	}
	return toClientShipment(sh), nil
}
