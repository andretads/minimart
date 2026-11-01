package application

import (
	"context"
	"minimart/internal/notification/domain"
)

// MarkNotificationUnread marks a single notification as unread for the given user.
func (s *Service) MarkNotificationUnread(ctx context.Context, id, userID string) error {
	if id == "" {
		return domain.ErrEmptyID
	}
	if userID == "" {
		return domain.ErrEmptyUserID
	}
	return s.repo.MarkUnread(ctx, id, userID)
}
