package application

import (
	"context"
	"minimart/internal/notification/domain"
)

func (s *Service) SendPaymentFailed(ctx context.Context, orderID, userID string, amountCents int64, reason string) error {
	n, err := domain.NewPaymentFailed(newID(), orderID, userID, amountCents, reason)
	_, _, err = s.IngestNotification(ctx, n, err)
	return err
}
