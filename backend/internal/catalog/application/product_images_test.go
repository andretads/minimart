package application

import (
	"bytes"
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

func TestProductImages(t *testing.T) {
	t.Run("Product does not exist", func(t *testing.T) {
		svc, _ := newTestService(t)

		img := &ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
		_, err := svc.AddProductImage(t.Context(), "missing", img)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Image is missing", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedImageProduct(t, repo)

		_, err := svc.AddProductImage(t.Context(), "p1", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrImageRequired)
	})

	t.Run("Unsupported image type", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedImageProduct(t, repo)

		img := &ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.txt", ContentType: "text/plain", Size: 10}
		_, err := svc.AddProductImage(t.Context(), "p1", img)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidImageFormat)
	})

	t.Run("Image is too large", func(t *testing.T) {
		svc, repo := newTestService(t)
		seedImageProduct(t, repo)

		img := &ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 6 * 1024 * 1024}
		_, err := svc.AddProductImage(t.Context(), "p1", img)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrImageTooLarge)
	})

	t.Run("Image does not belong to the product", func(t *testing.T) {
		repo := memory.New()
		// p1 and p2
		p1, _ := domain.NewProduct("p1", "Widget", 500, 3, "cat-1", "", "", nil)
		p2, _ := domain.NewProduct("p2", "Gadget", 500, 3, "cat-1", "", "", nil)
		require.NoError(t, repo.Save(t.Context(), p1))
		require.NoError(t, repo.Save(t.Context(), p2))
		svc := NewService(repo, &recordingS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		img := &ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
		saved, err := svc.AddProductImage(t.Context(), "p2", img)
		require.NoError(t, err)

		_, err = svc.DeleteProductImage(t.Context(), "p1", saved.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductImageNotFound)
	})

	t.Run("First image is added at position 0", func(t *testing.T) {
		repo := memory.New()
		seedImageProduct(t, repo)
		svc := NewService(repo, &recordingS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		img := &ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
		saved, err := svc.AddProductImage(t.Context(), "p1", img)
		require.NoError(t, err)
		assert.Equal(t, 0, saved.Position)

		p, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 2, p.Version)
	})

	t.Run("Additional images are appended", func(t *testing.T) {
		repo := memory.New()
		seedImageProduct(t, repo)
		svc := NewService(repo, &recordingS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		img := &ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
		_, err := svc.AddProductImage(t.Context(), "p1", img)
		require.NoError(t, err)

		img2 := &ImageUpload{Reader: bytes.NewReader([]byte("y")), Filename: "b.jpg", ContentType: "image/jpeg", Size: 10}
		saved2, err := svc.AddProductImage(t.Context(), "p1", img2)
		require.NoError(t, err)
		assert.Equal(t, 1, saved2.Position)

		p, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 3, p.Version)
	})

	t.Run("Product response includes its images", func(t *testing.T) {
		repo := memory.New()
		seedImageProduct(t, repo)
		svc := NewService(repo, &recordingS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		img := &ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
		_, err := svc.AddProductImage(t.Context(), "p1", img)
		require.NoError(t, err)

		images, err := svc.ListProductImages(t.Context(), "p1")
		require.NoError(t, err)
		assert.Len(t, images, 1)
	})

	t.Run("Image is deleted", func(t *testing.T) {
		repo := memory.New()
		seedImageProduct(t, repo)
		svc := NewService(repo, &recordingS3Client{}, fakeCategoryProvider{}, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider())

		img := &ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
		saved, err := svc.AddProductImage(t.Context(), "p1", img)
		require.NoError(t, err)

		_, err = svc.DeleteProductImage(t.Context(), "p1", saved.ID)
		require.NoError(t, err)

		images, err := svc.ListProductImages(t.Context(), "p1")
		require.NoError(t, err)
		assert.Empty(t, images)

		p, err := repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 3, p.Version)
	})
}
