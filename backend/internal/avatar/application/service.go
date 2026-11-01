package application

import (
	"context"
	"io"
	"minimart/internal/avatar/domain"
	"minimart/pkg/metrics"
	"time"
)

// S3Client defines the interface for S3 operations needed by the avatar service.
type S3Client interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string, size int64) error
	Delete(ctx context.Context, key string) error
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
	Bucket() string
}

// AvatarView is the avatar with its presigned URL for HTTP responses.
type AvatarView struct {
	Avatar domain.Avatar
	URL    string
}

// UploadAvatarRequest is the input for UploadAvatar.
type UploadAvatarRequest struct {
	AccountID string
	Image     *ImageUpload
}

// GetAvatarRequest is the input for GetAvatar.
type GetAvatarRequest struct {
	AccountID string
}

// Service is the avatar application layer. It orchestrates use cases over the
// domain and its ports, and depends only on interfaces - never on a concrete
// database or HTTP type.
type Service struct {
	repo         domain.Repository
	s3Client     S3Client
	presignTTL   time.Duration
	maxImageSize int64
	m            metrics.Metrics
}

// NewService injects the repository port, S3 client, and metrics recorder.
func NewService(repo domain.Repository, s3Client S3Client, presignTTL time.Duration, maxImageSize int64, m metrics.Metrics) *Service {
	return &Service{repo: repo, s3Client: s3Client, presignTTL: presignTTL, maxImageSize: maxImageSize, m: m}
}
