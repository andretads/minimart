package application

import (
	"context"
	"errors"
	"minimart/internal/avatar/domain"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAvatar(t *testing.T) {
	t.Run("Empty account ID is rejected", func(t *testing.T) {
		svc, _, _ := newTestService(t)

		_, err := svc.GetAvatar(t.Context(), GetAvatarRequest{AccountID: ""})
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrEmptyAccountID)
		assert.EqualError(t, err, "avatar: id da conta é obrigatório")
	})

	t.Run("Avatar not found", func(t *testing.T) {
		svc, _, _ := newTestService(t)

		_, err := svc.GetAvatar(t.Context(), GetAvatarRequest{AccountID: "missing-account"})
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrAvatarNotFound)
		assert.EqualError(t, err, "avatar: avatar não encontrado")
	})

	t.Run("Avatar returned with presigned URL", func(t *testing.T) {
		svc, _, _ := newTestService(t)
		createTestAvatar(t, svc, "acct-1")

		view, err := svc.GetAvatar(t.Context(), GetAvatarRequest{AccountID: "acct-1"})
		require.NoError(t, err)

		assert.Equal(t, "acct-1", view.Avatar.AccountID)
		assert.Equal(t, "image/jpeg", view.Avatar.ContentType)
		assert.Equal(t, int64(100), view.Avatar.SizeBytes)
		assert.NotEmpty(t, view.URL)
	})

	t.Run("Presign failure propagates", func(t *testing.T) {
		svc, _, repo := newTestService(t)
		createTestAvatar(t, svc, "acct-1")

		s3Client := &recordingS3Client{
			presignFn: func(ctx context.Context, key string, expiry time.Duration) (string, error) {
				return "", errors.New("presign unavailable")
			},
		}
		failSvc := NewService(repo, s3Client, time.Hour, testMaxImageSize, metrics.Noop{})

		_, err := failSvc.GetAvatar(t.Context(), GetAvatarRequest{AccountID: "acct-1"})
		require.Error(t, err)
	})
}
