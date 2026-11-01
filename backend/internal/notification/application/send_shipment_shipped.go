package application

import (
	"context"
	"minimart/internal/notification/domain"
)

func (s *Service) SendShipmentShipped(ctx context.Context, orderID, userID, trackingCode, carrier string) error {
	n, err := domain.NewShipmentShipped(newID(), orderID, userID, trackingCode, carrier)
	_, _, err = s.IngestNotification(ctx, n, err)
	return err
}
