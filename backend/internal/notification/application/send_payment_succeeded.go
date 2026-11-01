package application

import (
	"context"
	"minimart/internal/notification/domain"
)

func (s *Service) SendPaymentSucceeded(ctx context.Context, orderID, userID string, amountCents int64) error {
	n, err := domain.NewPaymentSucceeded(newID(), orderID, userID, amountCents)
	_, _, err = s.IngestNotification(ctx, n, err)
	return err
}
