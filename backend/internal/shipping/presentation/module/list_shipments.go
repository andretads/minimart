package module

import (
	"context"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// ListShipments returns a paginated list of shipments for a user.
func (s *Shipping) ListShipments(ctx context.Context, userID string, page, pageSize int) ([]shippingclient.Shipment, int, error) {
	shipments, total, err := s.svc.ListShipments(ctx, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	resp := make([]shippingclient.Shipment, len(shipments))
	for i, sh := range shipments {
		resp[i] = toClientShipment(sh)
	}
	return resp, total, nil
}
