package application

import (
	"context"
	"minimart/internal/notification/domain"
)

func (s *Service) ListDeliveries(ctx context.Context, notificationID, userID string) ([]domain.Delivery, error) {
	if notificationID == "" {
		return nil, domain.ErrEmptyNotificationID
	}
	n, err := s.repo.GetByIDForUser(ctx, notificationID, userID)
	if err != nil {
		return nil, err
	}
	return s.deliveries.ListByNotification(ctx, n.ID)
}
