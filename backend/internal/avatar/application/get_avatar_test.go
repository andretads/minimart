package application

import (
	"bytes"
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

func TestGetAvatar(t *testing.T) {
	repo := memory.New()
	svc := NewService(repo, &recordingS3Client{}, time.Hour, 5*1024*1024, metrics.Noop{})

	t.Run("returns avatar with presigned URL", func(t *testing.T) {
		_, _ = svc.UploadAvatar(t.Context(), UploadAvatarRequest{
			AccountID: "test-account-7",
			Image: &ImageUpload{
				Reader:      bytes.NewReader([]byte("data")),
				ContentType: "image/jpeg",
				Size:        100,
			},
		})

		view, err := svc.GetAvatar(t.Context(), GetAvatarRequest{
			AccountID: "test-account-7",
		})
		require.NoError(t, err)

		assert.Equal(t, "test-account-7", view.Avatar.AccountID)
		assert.NotEmpty(t, view.URL)
	})

	t.Run("returns ErrAvatarNotFound when missing", func(t *testing.T) {
		_, err := svc.GetAvatar(t.Context(), GetAvatarRequest{
			AccountID: "missing-account",
		})
		assert.ErrorIs(t, err, domain.ErrAvatarNotFound)
	})

	t.Run("rejects empty account id", func(t *testing.T) {
		_, err := svc.GetAvatar(t.Context(), GetAvatarRequest{
			AccountID: "",
		})
		assert.ErrorIs(t, err, domain.ErrEmptyAccountID)
	})

	t.Run("returns error on presign failure", func(t *testing.T) {
		repo := memory.New()
		okClient := &recordingS3Client{}
		okSvc := NewService(repo, okClient, time.Hour, 5*1024*1024, metrics.Noop{})
		_, err := okSvc.UploadAvatar(t.Context(), UploadAvatarRequest{
			AccountID: "test-account-8",
			Image: &ImageUpload{
				Reader:      bytes.NewReader([]byte("data")),
				ContentType: "image/jpeg",
				Size:        100,
			},
		})
		require.NoError(t, err)

		failClient := &recordingS3Client{}
		failClient.presignFn = func(ctx context.Context, key string, expiry time.Duration) (string, error) {
			return "", errors.New("presign failed")
		}
		failSvc := NewService(repo, failClient, time.Hour, 5*1024*1024, metrics.Noop{})

		_, err = failSvc.GetAvatar(t.Context(), GetAvatarRequest{
			AccountID: "test-account-8",
		})
		require.Error(t, err)
	})
}
