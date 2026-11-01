package application

import (
	"context"
	"minimart/internal/notification/domain"
)

func (s *Service) SendOrderConfirmation(ctx context.Context, orderID, userID string, totalCents int64) error {
	n, err := domain.NewOrderConfirmation(newID(), orderID, userID, totalCents)
	_, _, err = s.IngestNotification(ctx, n, err)
	return err
}
