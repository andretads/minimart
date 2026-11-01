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
