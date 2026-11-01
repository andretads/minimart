package application

import (
	"context"
	"errors"
	"io"
	"minimart/internal/avatar/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadAvatar(t *testing.T) {
	t.Run("Empty account ID is rejected", func(t *testing.T) {
		svc, _, _ := newTestService(t)

		req := validUploadAvatarRequest("")

		_, err := svc.UploadAvatar(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrEmptyAccountID)
		assert.EqualError(t, err, "avatar: id da conta é obrigatório")
	})

	t.Run("Image is required", func(t *testing.T) {
		svc, _, _ := newTestService(t)

		req := validUploadAvatarRequest("acct-1")
		req.Image = nil

		_, err := svc.UploadAvatar(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrImageRequired)
		assert.EqualError(t, err, "avatar: imagem é obrigatória")
	})

	t.Run("Invalid image format is rejected", func(t *testing.T) {
		svc, _, _ := newTestService(t)

		req := validUploadAvatarRequest("acct-1")
		req.Image = imageUpload("text/plain", 100)

		_, err := svc.UploadAvatar(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrInvalidImageFormat)
		assert.EqualError(t, err, "avatar: formato de imagem inválido, deve ser uma imagem")
	})

	t.Run("Invalid image size is rejected", func(t *testing.T) {
		svc, _, _ := newTestService(t)

		req := validUploadAvatarRequest("acct-1")
		req.Image = imageUpload("image/jpeg", 0)

		_, err := svc.UploadAvatar(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrInvalidImageSize)
		assert.EqualError(t, err, "avatar: tamanho de imagem inválido")
	})

	t.Run("Image too large is rejected", func(t *testing.T) {
		svc, _, _ := newTestService(t)

		req := validUploadAvatarRequest("acct-1")
		req.Image = imageUpload("image/jpeg", 6291456)

		_, err := svc.UploadAvatar(t.Context(), req)
		require.Error(t, err)
		require.ErrorIs(t, err, domain.ErrImageTooLarge)
		assert.EqualError(t, err, "avatar: imagem excede o tamanho máximo permitido")
	})

	t.Run("Upload failure propagates", func(t *testing.T) {
		svc, s3Client, repo := newTestService(t)
		boom := errors.New("s3 unavailable")
		s3Client.uploadFn = func(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
			return boom
		}

		_, err := svc.UploadAvatar(t.Context(), validUploadAvatarRequest("acct-1"))
		require.Error(t, err)
		require.ErrorIs(t, err, boom)
		assert.Contains(t, err.Error(), "upload avatar")

		_, getErr := repo.GetByAccountID(t.Context(), "acct-1")
		assert.ErrorIs(t, getErr, domain.ErrAvatarNotFound)
	})

	t.Run("Image uploaded and metadata saved", func(t *testing.T) {
		svc, s3Client, repo := newTestService(t)

		view, err := svc.UploadAvatar(t.Context(), validUploadAvatarRequest("acct-1"))
		require.NoError(t, err)

		assert.Equal(t, "acct-1", view.Avatar.AccountID)
		assert.Equal(t, "avatars/acct-1/avatar", view.Avatar.ObjectKey)
		assert.Equal(t, "test-bucket", view.Avatar.Bucket)
		assert.Equal(t, "image/jpeg", view.Avatar.ContentType)
		assert.Equal(t, int64(100), view.Avatar.SizeBytes)
		assert.Equal(t, 1, view.Avatar.Version)
		assert.NotEmpty(t, view.URL)

		require.Len(t, s3Client.uploads, 1)
		assert.Equal(t, "avatars/acct-1/avatar", s3Client.uploads[0].key)
		assert.Equal(t, "image/jpeg", s3Client.uploads[0].contentType)
		assert.Equal(t, int64(100), s3Client.uploads[0].size)

		stored, err := repo.GetByAccountID(t.Context(), "acct-1")
		require.NoError(t, err)
		assert.Equal(t, view.Avatar.AccountID, stored.AccountID)
		assert.Equal(t, view.Avatar.ObjectKey, stored.ObjectKey)
		assert.Equal(t, view.Avatar.Bucket, stored.Bucket)
		assert.Equal(t, view.Avatar.ContentType, stored.ContentType)
		assert.Equal(t, view.Avatar.SizeBytes, stored.SizeBytes)
		assert.Equal(t, view.Avatar.Version, stored.Version)
	})

	t.Run("Existing avatar overwritten with version bump", func(t *testing.T) {
		svc, _, repo := newTestService(t)

		createTestAvatar(t, svc, "acct-1")
		req := validUploadAvatarRequest("acct-1")
		req.Image = imageUpload("image/png", 200)

		view, err := svc.UploadAvatar(t.Context(), req)
		require.NoError(t, err)

		assert.Equal(t, "acct-1", view.Avatar.AccountID)
		assert.Equal(t, "image/png", view.Avatar.ContentType)
		assert.Equal(t, int64(200), view.Avatar.SizeBytes)
		assert.Equal(t, 2, view.Avatar.Version)

		stored, err := repo.GetByAccountID(t.Context(), "acct-1")
		require.NoError(t, err)
		assert.Equal(t, "image/png", stored.ContentType)
		assert.Equal(t, int64(200), stored.SizeBytes)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Object key follows the avatars/{account_id}/avatar convention", func(t *testing.T) {
		svc, s3Client, repo := newTestService(t)

		req := validUploadAvatarRequest("acct-2")
		req.Image = imageUpload("image/webp", 50)

		_, err := svc.UploadAvatar(t.Context(), req)
		require.NoError(t, err)

		require.Len(t, s3Client.uploads, 1)
		assert.Equal(t, "avatars/acct-2/avatar", s3Client.uploads[0].key)

		stored, err := repo.GetByAccountID(t.Context(), "acct-2")
		require.NoError(t, err)
		assert.Equal(t, "avatars/acct-2/avatar", stored.ObjectKey)
	})

	t.Run("Supported image types", func(t *testing.T) {
		svc, _, _ := newTestService(t)

		req := validUploadAvatarRequest("acct-3")
		req.Image = imageUpload("image/gif", 50)

		view, err := svc.UploadAvatar(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "image/gif", view.Avatar.ContentType)
	})
}
