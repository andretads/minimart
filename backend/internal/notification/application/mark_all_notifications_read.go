package application

import (
	"context"
	"minimart/internal/notification/domain"
)

// MarkAllNotificationsRead marks every notification of the user as read.
func (s *Service) MarkAllNotificationsRead(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrEmptyUserID
	}
	return s.repo.MarkAllRead(ctx, userID)
}
