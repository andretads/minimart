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

// MarkAllNotificationsRead marks every notification of the user as read.
func (s *Service) MarkAllNotificationsRead(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrEmptyUserID
	}
	return s.repo.MarkAllRead(ctx, userID)
}

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
