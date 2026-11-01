package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"minimart/internal/avatar/domain"
	"minimart/internal/avatar/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadAvatar(t *testing.T) {
	svc, s3Client, _ := newTestService(t, 5*1024*1024)
	require.NotEqual(t, (*recordingS3Client)(nil), s3Client)

	t.Run("uploads image and saves metadata", func(t *testing.T) {
		img := &ImageUpload{
			Reader:      bytes.NewReader([]byte("fake image data")),
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
			Size:        100,
		}

		view, err := svc.UploadAvatar(t.Context(), UploadAvatarRequest{
			AccountID: "test-account-1",
			Image:     img,
		})
		require.NoError(t, err)

		assert.Equal(t, "test-account-1", view.Avatar.AccountID)
		assert.Equal(t, "avatars/test-account-1/avatar", view.Avatar.ObjectKey)
		assert.Equal(t, "test-bucket", view.Avatar.Bucket)
		assert.Equal(t, "image/jpeg", view.Avatar.ContentType)
		assert.Equal(t, int64(100), view.Avatar.SizeBytes)
		assert.Equal(t, 1, view.Avatar.Version)
		assert.NotEmpty(t, view.URL)
		assert.Len(t, s3Client.uploads, 1)
		assert.Equal(t, "avatars/test-account-1/avatar", s3Client.uploads[0].key)
		assert.Equal(t, "image/jpeg", s3Client.uploads[0].contentType)
		assert.Equal(t, int64(100), s3Client.uploads[0].size)
		assert.Equal(t, []byte("fake image data"), s3Client.uploads[0].body)
	})

	t.Run("rejects empty account id", func(t *testing.T) {
		_, err := svc.UploadAvatar(t.Context(), UploadAvatarRequest{
			AccountID: "",
			Image:     &ImageUpload{ContentType: "image/jpeg", Size: 100},
		})
		assert.ErrorIs(t, err, domain.ErrEmptyAccountID)
	})

	t.Run("rejects nil image", func(t *testing.T) {
		_, err := svc.UploadAvatar(t.Context(), UploadAvatarRequest{
			AccountID: "test-account-2",
			Image:     nil,
		})
		assert.ErrorIs(t, err, domain.ErrImageRequired)
	})

	t.Run("rejects invalid content type", func(t *testing.T) {
		_, err := svc.UploadAvatar(t.Context(), UploadAvatarRequest{
			AccountID: "test-account-3",
			Image: &ImageUpload{
				ContentType: "text/plain",
				Size:        100,
			},
		})
		assert.ErrorIs(t, err, domain.ErrInvalidImageFormat)
	})

	t.Run("rejects too large image", func(t *testing.T) {
		_, err := svc.UploadAvatar(t.Context(), UploadAvatarRequest{
			AccountID: "test-account-4",
			Image: &ImageUpload{
				ContentType: "image/jpeg",
				Size:        6 * 1024 * 1024,
			},
		})
		assert.ErrorIs(t, err, domain.ErrImageTooLarge)
	})

	t.Run("wraps upload errors", func(t *testing.T) {
		boom := errors.New("network down")
		failClient := &recordingS3Client{}
		failClient.uploadFn = func(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
			return boom
		}
		svc := NewService(memory.New(), failClient, time.Hour, 5*1024*1024, metrics.Noop{})

		_, err := svc.UploadAvatar(t.Context(), UploadAvatarRequest{
			AccountID: "test-account-5",
			Image: &ImageUpload{
				Reader:      bytes.NewReader([]byte("data")),
				ContentType: "image/jpeg",
				Size:        10,
			},
		})
		require.ErrorIs(t, err, boom)
		assert.Contains(t, err.Error(), "upload avatar")
	})

	t.Run("overwrites existing avatar and increments version", func(t *testing.T) {
		repo := memory.New()
		svc := NewService(repo, &recordingS3Client{}, time.Hour, 5*1024*1024, metrics.Noop{})

		img := &ImageUpload{
			Reader:      bytes.NewReader([]byte("v1")),
			ContentType: "image/jpeg",
			Size:        100,
		}
		_, err := svc.UploadAvatar(t.Context(), UploadAvatarRequest{
			AccountID: "test-account-6",
			Image:     img,
		})
		require.NoError(t, err)

		img2 := &ImageUpload{
			Reader:      bytes.NewReader([]byte("v2")),
			ContentType: "image/png",
			Size:        200,
		}
		view, err := svc.UploadAvatar(t.Context(), UploadAvatarRequest{
			AccountID: "test-account-6",
			Image:     img2,
		})
		require.NoError(t, err)

		assert.Equal(t, 2, view.Avatar.Version)
		assert.Equal(t, "image/png", view.Avatar.ContentType)
		assert.Equal(t, int64(200), view.Avatar.SizeBytes)
	})
}
