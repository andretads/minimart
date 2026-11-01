package application

import (
	"context"
	"io"
	"minimart/internal/avatar/domain"
	"minimart/pkg/logger"
)

// ImageUpload is an inbound image stream supplied to a use case.
type ImageUpload struct {
	Reader      io.Reader
	Filename    string
	ContentType string
	Size        int64
}

func (s *Service) validateImage(img *ImageUpload) error {
	if img == nil {
		return domain.ErrImageRequired
	}
	if !domain.IsSupportedImageType(img.ContentType) {
		return domain.ErrInvalidImageFormat
	}
	if img.Size <= 0 {
		return domain.ErrInvalidImageSize
	}
	if img.Size > s.maxImageSize {
		return domain.ErrImageTooLarge
	}
	return nil
}

func avatarKey(accountID string) string {
	return "avatars/" + accountID + "/avatar"
}

func (s *Service) presignURL(ctx context.Context, key string) (string, error) {
	if key == "" || s.s3Client == nil {
		return "", nil
	}
	url, err := s.s3Client.PresignGet(ctx, key, s.presignTTL)
	if err != nil {
		logger.FromContext(ctx).Warn("avatar presign failed",
			"operation", "s3_presign",
			"key", key,
			"bucket", s.s3Client.Bucket(),
			"error", err,
		)
		return "", err
	}
	return url, nil
}
