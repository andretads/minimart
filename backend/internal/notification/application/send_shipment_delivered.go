package application

import (
	"context"
	"minimart/internal/notification/domain"
)

func (s *Service) SendShipmentDelivered(ctx context.Context, orderID, userID string) error {
	n, err := domain.NewShipmentDelivered(newID(), orderID, userID)
	_, _, err = s.IngestNotification(ctx, n, err)
	return err
}
