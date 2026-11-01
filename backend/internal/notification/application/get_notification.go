package application

import (
	"context"
	"minimart/internal/notification/domain"
)

func (s *Service) GetNotification(ctx context.Context, id, userID string) (domain.Notification, error) {
	if id == "" {
		return domain.Notification{}, domain.ErrEmptyID
	}
	if userID == "" {
		return domain.Notification{}, domain.ErrEmptyUserID
	}
	return s.repo.GetByIDForUser(ctx, id, userID)
}

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
