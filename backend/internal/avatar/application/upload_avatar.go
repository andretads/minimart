package application

import (
	"context"
	"fmt"
	"minimart/internal/avatar/domain"
	"minimart/pkg/logger"
	"minimart/pkg/metrics"
)

// UploadAvatar uploads an avatar image and persists the metadata.
func (s *Service) UploadAvatar(ctx context.Context, req UploadAvatarRequest) (AvatarView, error) {
	if req.AccountID == "" {
		s.m.IncAvatarUploaded(metrics.StatusError)
		return AvatarView{}, domain.ErrEmptyAccountID
	}

	if req.Image == nil {
		s.m.IncAvatarUploaded(metrics.StatusError)
		return AvatarView{}, domain.ErrImageRequired
	}

	if err := s.validateImage(req.Image); err != nil {
		s.m.IncAvatarUploaded(metrics.StatusError)
		return AvatarView{}, err
	}

	key := avatarKey(req.AccountID)
	if err := s.s3Client.Upload(ctx, key, req.Image.Reader, req.Image.ContentType, req.Image.Size); err != nil {
		logger.FromContext(ctx).Error("avatar upload failed",
			"operation", "s3_upload",
			"account_id", req.AccountID,
			"key", key,
			"bucket", s.s3Client.Bucket(),
			"content_type", req.Image.ContentType,
			"size_bytes", req.Image.Size,
			"error", err,
		)
		s.m.IncAvatarUploaded(metrics.StatusError)
		return AvatarView{}, fmt.Errorf("upload avatar: %w", err)
	}

	avatar, err := domain.NewAvatar(req.AccountID, key, s.s3Client.Bucket(), req.Image.ContentType, req.Image.Size)
	if err != nil {
		if deleteErr := s.s3Client.Delete(ctx, key); deleteErr != nil {
			logger.FromContext(ctx).Warn("avatar cleanup failed",
				"operation", "s3_delete",
				"account_id", req.AccountID,
				"key", key,
				"error", deleteErr,
			)
		}
		s.m.IncAvatarUploaded(metrics.StatusError)
		return AvatarView{}, err
	}

	saved, err := s.repo.Save(ctx, avatar)
	if err != nil {
		logger.FromContext(ctx).Error("avatar upload failed",
			"operation", "db_save",
			"account_id", req.AccountID,
			"key", key,
			"error", err,
		)
		if deleteErr := s.s3Client.Delete(ctx, key); deleteErr != nil {
			logger.FromContext(ctx).Warn("avatar cleanup failed",
				"operation", "s3_delete",
				"account_id", req.AccountID,
				"key", key,
				"error", deleteErr,
			)
		}
		s.m.IncAvatarUploaded(metrics.StatusError)
		return AvatarView{}, err
	}

	url, err := s.presignURL(ctx, key)
	if err != nil {
		s.m.IncAvatarUploaded(metrics.StatusError)
		return AvatarView{}, fmt.Errorf("presign avatar: %w", err)
	}
	s.m.IncAvatarUploaded(metrics.StatusSuccess)
	return AvatarView{Avatar: saved, URL: url}, nil
}
