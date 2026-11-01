package application

import (
	"context"
	"minimart/internal/notification/domain"
)

func (s *Service) SendShipmentCreated(ctx context.Context, orderID, userID string, shippingCostCents int64) error {
	n, err := domain.NewShipmentCreated(newID(), orderID, userID, shippingCostCents)
	_, _, err = s.IngestNotification(ctx, n, err)
	return err
}
