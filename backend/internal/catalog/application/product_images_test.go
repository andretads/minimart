package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/memory"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedImageProduct(t *testing.T, repo *memory.Repository) {
	t.Helper()
	p, err := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))
}

func TestAddProductImage(t *testing.T) {
	repo := memory.New()
	seedImageProduct(t, repo)
	s3 := &recordingS3Client{}
	svc := NewService(repo, s3, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

	t.Run("adds an image successfully", func(t *testing.T) {
		img := &ImageUpload{
			Reader:      bytes.NewReader([]byte("fake")),
			Filename:    "test.jpg",
			ContentType: "image/jpeg",
			Size:        100,
		}
		saved, err := svc.AddProductImage(t.Context(), "p1", img)
		require.NoError(t, err)
		assert.Equal(t, "p1", saved.ProductID)
		assert.Equal(t, "test-bucket", saved.Bucket)
		assert.NotEmpty(t, saved.ID)
		assert.Len(t, s3.uploads, 1)
	})

	t.Run("product not found", func(t *testing.T) {
		img := &ImageUpload{ContentType: "image/jpeg", Size: 10}
		_, err := svc.AddProductImage(t.Context(), "missing", img)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("nil image", func(t *testing.T) {
		_, err := svc.AddProductImage(t.Context(), "p1", nil)
		assert.ErrorIs(t, err, domain.ErrImageRequired)
	})

	t.Run("invalid image format", func(t *testing.T) {
		img := &ImageUpload{ContentType: "text/plain", Size: 10}
		_, err := svc.AddProductImage(t.Context(), "p1", img)
		assert.ErrorIs(t, err, domain.ErrInvalidImageFormat)
	})

	t.Run("storage failure cleans up s3", func(t *testing.T) {
		repo2 := memory.New()
		seedImageProduct(t, repo2)
		s3Fail := &recordingS3Client{}
		s3Fail.uploadFn = func(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
			return errors.New("upload failed")
		}
		svc2 := NewService(repo2, s3Fail, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())
		img := &ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
		_, err := svc2.AddProductImage(t.Context(), "p1", img)
		assert.Error(t, err)
	})
}

func TestListProductImages(t *testing.T) {
	repo := memory.New()
	seedImageProduct(t, repo)
	svc := NewService(repo, &recordingS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

	img := &ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
	saved, err := svc.AddProductImage(t.Context(), "p1", img)
	require.NoError(t, err)

	images, err := svc.ListProductImages(t.Context(), "p1")
	require.NoError(t, err)
	require.Len(t, images, 1)
	assert.Equal(t, saved.ID, images[0].ID)

	_, err = svc.ListProductImages(t.Context(), "missing")
	assert.ErrorIs(t, err, domain.ErrProductNotFound)
}

func TestDeleteProductImage(t *testing.T) {
	repo := memory.New()
	seedImageProduct(t, repo)
	s3 := &recordingS3Client{}
	svc := NewService(repo, s3, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

	img := &ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
	saved, err := svc.AddProductImage(t.Context(), "p1", img)
	require.NoError(t, err)

	deleted, err := svc.DeleteProductImage(t.Context(), "p1", saved.ID)
	require.NoError(t, err)
	assert.Equal(t, saved.ID, deleted.ID)
	assert.Len(t, s3.deletes, 1)

	_, err = svc.DeleteProductImage(t.Context(), "p1", "nope")
	assert.ErrorIs(t, err, domain.ErrProductImageNotFound)
}
