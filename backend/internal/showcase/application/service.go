package application

import (
	"context"
	"minimart/internal/showcase/domain"
	"time"
)

// ImageURLSigner generates short-lived URLs for stored image objects.
type ImageURLSigner interface {
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// Service orchestrates the showcase use cases.
type Service struct {
	repo       domain.Repository
	signer     ImageURLSigner
	presignTTL time.Duration
}

func NewService(repo domain.Repository, signer ImageURLSigner, presignTTL time.Duration) *Service {
	return &Service{repo: repo, signer: signer, presignTTL: presignTTL}
}

// PresignImageURL generates a presigned URL for a stored image object. It
// returns an empty string when the key is empty, no signer is configured, or
// signing fails.
func (s *Service) PresignImageURL(ctx context.Context, key string) string {
	if key == "" || s.signer == nil {
		return ""
	}
	url, err := s.signer.PresignGet(ctx, key, s.presignTTL)
	if err != nil {
		return ""
	}
	return url
}
