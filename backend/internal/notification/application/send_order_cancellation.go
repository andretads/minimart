package application

import (
	"context"
	"minimart/internal/notification/domain"
)

func (s *Service) SendOrderCancellation(ctx context.Context, orderID, userID string, totalCents int64) error {
	n, err := domain.NewOrderCancellation(newID(), orderID, userID, totalCents)
	_, _, err = s.IngestNotification(ctx, n, err)
	return err
}
