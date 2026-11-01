package application

import (
	"context"
	"minimart/internal/notification/domain"
)

// MarkNotificationRead marks a single notification as read for the given user.
func (s *Service) MarkNotificationRead(ctx context.Context, id, userID string) error {
	if id == "" {
		return domain.ErrEmptyID
	}
	if userID == "" {
		return domain.ErrEmptyUserID
	}
	return s.repo.MarkRead(ctx, id, userID)
}
