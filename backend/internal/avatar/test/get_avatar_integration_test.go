//go:build integration

package test

import (
	"errors"
	"minimart/internal/avatar/application"
	"minimart/internal/avatar/domain"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAvatarIntegration(t *testing.T) {
	h := newAvatarHarness(t)

	t.Run("Empty account ID is rejected", func(t *testing.T) {
		_, err := h.svc.GetAvatar(t.Context(), application.GetAvatarRequest{AccountID: ""})
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrEmptyAccountID)
		assert.EqualError(t, err, "avatar: id da conta é obrigatório")
	})

	t.Run("Avatar not found", func(t *testing.T) {
		_, err := h.svc.GetAvatar(t.Context(), application.GetAvatarRequest{AccountID: "missing-account"})
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrAvatarNotFound)
		assert.EqualError(t, err, "avatar: avatar não encontrado")
	})

	t.Run("Avatar returned with presigned URL", func(t *testing.T) {
		accountID := newTestAccount(t, h.write)
		createTestAvatar(t, h.svc, accountID)

		view := getAvatarHTTP(t, h, accountID)

		assert.Equal(t, accountID, view.AccountID)
		assert.Equal(t, "image/jpeg", view.ContentType)
		assert.Equal(t, int64(100), view.SizeBytes)
		assert.NotEmpty(t, view.ImageURL)
	})

	t.Run("Presign failure propagates", func(t *testing.T) {
		accountID := newTestAccount(t, h.write)
		createTestAvatar(t, h.svc, accountID)

		svc := application.NewService(
			h.repo,
			&failingS3Client{presignErr: errors.New("presign unavailable")},
			time.Hour,
			testMaxImageSize,
			metrics.Noop{},
		)

		_, err := svc.GetAvatar(t.Context(), application.GetAvatarRequest{AccountID: accountID})
		require.Error(t, err)
	})
}
