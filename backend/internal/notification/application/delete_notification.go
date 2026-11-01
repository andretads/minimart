package application

import (
	"context"
	"minimart/internal/notification/domain"
)

// DeleteNotification removes a notification for the given user.
func (s *Service) DeleteNotification(ctx context.Context, id, userID string) error {
	if id == "" {
		return domain.ErrEmptyID
	}
	if userID == "" {
		return domain.ErrEmptyUserID
	}
	return s.repo.Delete(ctx, id, userID)
}
