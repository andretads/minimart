package application

import (
	"context"
	"io"
	"minimart/internal/catalog/domain"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"time"

	categoryclient "minimart/internal/category/presentation/module/client"
)

// S3Client defines the interface for S3 operations needed by the catalog service.
type S3Client interface {
	Upload(ctx context.Context, key string, body io.Reader, contentType string, size int64) error
	Delete(ctx context.Context, key string) error
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
	Bucket() string
}

// CategoryProvider is the interface for looking up categories.
// Implemented by the category contract adapter.
type CategoryProvider interface {
	GetCategory(ctx context.Context, id string) (categoryclient.CategoryDTO, error)
}

// Service is the catalog application layer. It orchestrates use cases over the
// domain and its ports, and depends only on interfaces - never on a concrete
// database or HTTP type.
type Service struct {
	repo             domain.Repository
	s3Client         S3Client
	categoryProvider CategoryProvider
	presignTTL       time.Duration
	maxImageSize     int64
	m                metrics.Metrics
	moderationSvc    *ModerationService
}

// NewService injects the repository port, S3 client, category provider, and the metrics recorder.
func NewService(repo domain.Repository, s3Client S3Client, categoryProvider CategoryProvider, presignTTL time.Duration, maxImageSize int64, m metrics.Metrics, moderationProvider moderation.Provider) *Service {
	svc := &Service{
		repo:             repo,
		s3Client:         s3Client,
		categoryProvider: categoryProvider,
		presignTTL:       presignTTL,
		maxImageSize:     maxImageSize,
		m:                m,
	}
	if moderationProvider != nil {
		svc.moderationSvc = NewModerationService(repo, moderationProvider, m)
	}
	return svc
}

// PresignImageURL generates a presigned URL for a stored image object.
// Returns an empty string when the key is empty or presigning fails.
func (s *Service) PresignImageURL(ctx context.Context, key string) string {
	if key == "" || s.s3Client == nil {
		return ""
	}
	url, err := s.s3Client.PresignGet(ctx, key, s.presignTTL)
	if err != nil {
		return ""
	}
	return url
}

// ModerateReview triggers moderation for a review.
func (s *Service) ModerateReview(ctx context.Context, req ModerateReviewRequest) error {
	if s.moderationSvc == nil {
		return domain.ErrModerationFailed
	}
	return s.moderationSvc.ModerateReview(ctx, req)
}
