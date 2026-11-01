package application

import (
	"context"
	"errors"
	"minimart/internal/avatar/domain"
	"minimart/internal/avatar/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateImage(t *testing.T) {
	svc, _, _ := newTestService(t, 5*1024*1024)

	tests := []struct {
		name    string
		img     *ImageUpload
		wantErr error
	}{
		{"nil image", nil, domain.ErrImageRequired},
		{"valid jpeg", &ImageUpload{ContentType: "image/jpeg", Size: 100}, nil},
		{"valid png", &ImageUpload{ContentType: "image/png", Size: 100}, nil},
		{"valid webp", &ImageUpload{ContentType: "image/webp", Size: 100}, nil},
		{"valid gif", &ImageUpload{ContentType: "image/gif", Size: 100}, nil},
		{"invalid content type", &ImageUpload{ContentType: "text/plain", Size: 100}, domain.ErrInvalidImageFormat},
		{"zero size", &ImageUpload{ContentType: "image/jpeg", Size: 0}, domain.ErrInvalidImageSize},
		{"negative size", &ImageUpload{ContentType: "image/jpeg", Size: -1}, domain.ErrInvalidImageSize},
		{"too large", &ImageUpload{ContentType: "image/jpeg", Size: 6 * 1024 * 1024}, domain.ErrImageTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateImage(tt.img)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAvatarKey(t *testing.T) {
	assert.Equal(t, "avatars/test-account/avatar", avatarKey("test-account"))
}

func TestPresignURL(t *testing.T) {
	repo := memory.New()
	svc := NewService(repo, &recordingS3Client{}, time.Hour, 5*1024*1024, metrics.Noop{})

	t.Run("returns presigned URL for valid key", func(t *testing.T) {
		url, err := svc.presignURL(t.Context(), "test-key")
		require.NoError(t, err)
		assert.Equal(t, "https://presigned.example.com/test-key", url)
	})

	t.Run("returns empty string for empty key", func(t *testing.T) {
		url, err := svc.presignURL(t.Context(), "")
		require.NoError(t, err)
		assert.Empty(t, url)
	})

	t.Run("returns error on presign failure", func(t *testing.T) {
		failClient := &recordingS3Client{}
		failClient.presignFn = func(ctx context.Context, key string, expiry time.Duration) (string, error) {
			return "", errors.New("presign failed")
		}
		svc := NewService(memory.New(), failClient, time.Hour, 5*1024*1024, metrics.Noop{})
		_, err := svc.presignURL(t.Context(), "test-key")
		require.Error(t, err)
	})
}
