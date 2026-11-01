package application

import (
	"context"
	"fmt"
	"minimart/internal/avatar/domain"
	"minimart/pkg/logger"
	"minimart/pkg/metrics"
)

// GetAvatar retrieves an avatar by account ID and returns it with a presigned URL.
func (s *Service) GetAvatar(ctx context.Context, req GetAvatarRequest) (AvatarView, error) {
	if req.AccountID == "" {
		s.m.IncAvatarFetched(metrics.StatusError)
		return AvatarView{}, domain.ErrEmptyAccountID
	}

	avatar, err := s.repo.GetByAccountID(ctx, req.AccountID)
	if err != nil {
		logger.FromContext(ctx).Error("avatar fetch failed",
			"operation", "db_get",
			"account_id", req.AccountID,
			"error", err,
		)
		s.m.IncAvatarFetched(metrics.StatusError)
		return AvatarView{}, err
	}

	url, err := s.presignURL(ctx, avatar.ObjectKey)
	if err != nil {
		s.m.IncAvatarFetched(metrics.StatusError)
		return AvatarView{}, fmt.Errorf("presign avatar: %w", err)
	}
	s.m.IncAvatarFetched(metrics.StatusSuccess)
	return AvatarView{Avatar: avatar, URL: url}, nil
}
