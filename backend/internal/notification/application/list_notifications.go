package application

import (
	"context"
	"minimart/internal/notification/domain"
)

func (s *Service) ListNotifications(ctx context.Context, userID string, page, pageSize int) ([]domain.Notification, int, error) {
	if userID == "" {
		return nil, 0, domain.ErrEmptyUserID
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	items, err := s.repo.ListByUserID(ctx, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
