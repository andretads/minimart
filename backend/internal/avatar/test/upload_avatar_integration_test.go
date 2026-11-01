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

func TestUploadAvatarIntegration(t *testing.T) {
	h := newAvatarHarness(t)

	t.Run("Empty account ID is rejected", func(t *testing.T) {
		req := validUploadAvatarRequest("")

		_, err := h.svc.UploadAvatar(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrEmptyAccountID)
		assert.EqualError(t, err, "avatar: id da conta é obrigatório")
	})

	t.Run("Image is required", func(t *testing.T) {
		req := validUploadAvatarRequest(newTestAccount(t, h.write))
		req.Image = nil

		_, err := h.svc.UploadAvatar(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrImageRequired)
		assert.EqualError(t, err, "avatar: imagem é obrigatória")
	})

	t.Run("Invalid image format is rejected", func(t *testing.T) {
		req := validUploadAvatarRequest(newTestAccount(t, h.write))
		req.Image = imageUpload("text/plain", 100)

		_, err := h.svc.UploadAvatar(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrInvalidImageFormat)
		assert.EqualError(t, err, "avatar: formato de imagem inválido, deve ser uma imagem")
	})

	t.Run("Invalid image size is rejected", func(t *testing.T) {
		req := validUploadAvatarRequest(newTestAccount(t, h.write))
		req.Image = imageUpload("image/jpeg", 0)

		_, err := h.svc.UploadAvatar(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrInvalidImageSize)
		assert.EqualError(t, err, "avatar: tamanho de imagem inválido")
	})

	t.Run("Image too large is rejected", func(t *testing.T) {
		req := validUploadAvatarRequest(newTestAccount(t, h.write))
		req.Image = imageUpload("image/jpeg", 6291456)

		_, err := h.svc.UploadAvatar(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrImageTooLarge)
		assert.EqualError(t, err, "avatar: imagem excede o tamanho máximo permitido")
	})

	t.Run("Upload failure propagates", func(t *testing.T) {
		accountID := newTestAccount(t, h.write)
		boom := errors.New("s3 unavailable")
		svc := application.NewService(h.repo, &failingS3Client{uploadErr: boom}, time.Hour, testMaxImageSize, metrics.Noop{})

		_, err := svc.UploadAvatar(t.Context(), validUploadAvatarRequest(accountID))
		require.Error(t, err)
		require.ErrorIs(t, err, boom)
		assert.Contains(t, err.Error(), "upload avatar")

		_, getErr := h.repo.GetByAccountID(t.Context(), accountID)
		assert.ErrorIs(t, getErr, domain.ErrAvatarNotFound)
	})

	t.Run("Image uploaded and metadata saved", func(t *testing.T) {
		accountID := newTestAccount(t, h.write)

		view := uploadAvatarHTTP(t, h, accountID, "image/jpeg", 100)

		assert.Equal(t, accountID, view.AccountID)
		assert.Equal(t, "image/jpeg", view.ContentType)
		assert.Equal(t, int64(100), view.SizeBytes)
		assert.Equal(t, 1, view.Version)
		assert.NotEmpty(t, view.ImageURL)

		stored, err := h.repo.GetByAccountID(t.Context(), accountID)
		require.NoError(t, err)
		assert.Equal(t, accountID, stored.AccountID)
		assert.Equal(t, "avatars/"+accountID+"/avatar", stored.ObjectKey)
		// The feature fixture "test-bucket" maps to the real test bucket.
		assert.Equal(t, testS3.Bucket, stored.Bucket)
		assert.Equal(t, "image/jpeg", stored.ContentType)
		assert.Equal(t, int64(100), stored.SizeBytes)
		assert.Equal(t, 1, stored.Version)
	})

	t.Run("Existing avatar overwritten with version bump", func(t *testing.T) {
		accountID := newTestAccount(t, h.write)
		first := uploadAvatarHTTP(t, h, accountID, "image/jpeg", 100)

		view := uploadAvatarHTTP(t, h, accountID, "image/png", 200)

		assert.Equal(t, accountID, view.AccountID)
		assert.Equal(t, "image/png", view.ContentType)
		assert.Equal(t, int64(200), view.SizeBytes)
		assert.Equal(t, 2, view.Version)
		assert.WithinDuration(t, first.CreatedAt, view.CreatedAt, time.Millisecond)

		stored, err := h.repo.GetByAccountID(t.Context(), accountID)
		require.NoError(t, err)
		assert.Equal(t, "image/png", stored.ContentType)
		assert.Equal(t, int64(200), stored.SizeBytes)
		assert.Equal(t, 2, stored.Version)
		assert.WithinDuration(t, first.CreatedAt, stored.CreatedAt, time.Millisecond)
	})

	t.Run("Object key follows the avatars/{account_id}/avatar convention", func(t *testing.T) {
		accountID := newTestAccount(t, h.write)

		req := validUploadAvatarRequest(accountID)
		req.Image = imageUpload("image/webp", 50)

		_, err := h.svc.UploadAvatar(t.Context(), req)
		require.NoError(t, err)

		stored, err := h.repo.GetByAccountID(t.Context(), accountID)
		require.NoError(t, err)
		assert.Equal(t, "avatars/"+accountID+"/avatar", stored.ObjectKey)
	})

	t.Run("Supported image types", func(t *testing.T) {
		accountID := newTestAccount(t, h.write)

		req := validUploadAvatarRequest(accountID)
		req.Image = imageUpload("image/gif", 50)

		view, err := h.svc.UploadAvatar(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "image/gif", view.Avatar.ContentType)
	})
}
