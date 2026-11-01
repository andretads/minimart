package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// GetShipmentByOrderID returns a shipment by order ID.
func (s *Shipping) GetShipmentByOrderID(ctx context.Context, orderID string) (shippingclient.Shipment, error) {
	sh, err := s.svc.GetShipmentByOrderID(ctx, orderID)
	if err != nil {
		if errors.Is(err, domain.ErrShipmentNotFound) {
			return shippingclient.Shipment{}, shippingclient.ErrShipmentNotFound
		}
		return shippingclient.Shipment{}, err
	}
	return toClientShipment(sh), nil
}
