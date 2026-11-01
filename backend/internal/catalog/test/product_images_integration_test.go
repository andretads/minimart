//go:build integration

package test

import (
	"bytes"
	"errors"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/postgres"
	"minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductImagesIntegration(t *testing.T) {
	env := newCatalogEnv(t)

	t.Run("Product does not exist", func(t *testing.T) {
		img := &application.ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
		_, err := env.Svc.AddProductImage(t.Context(), "missing", img)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Image is missing", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		_, err := env.Svc.AddProductImage(t.Context(), "p1", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrImageRequired)
	})

	t.Run("Unsupported image type", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		img := &application.ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.txt", ContentType: "text/plain", Size: 10}
		_, err := env.Svc.AddProductImage(t.Context(), "p1", img)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidImageFormat)
	})

	t.Run("Image is too large", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		img := &application.ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 6 * 1024 * 1024}
		_, err := env.Svc.AddProductImage(t.Context(), "p1", img)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrImageTooLarge)
	})

	t.Run("Image does not belong to the product", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		seedCatalogProduct(t, env, "p2", 3)
		img := &application.ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
		saved, err := env.Svc.AddProductImage(t.Context(), "p2", img)
		require.NoError(t, err)

		_, err = env.Svc.DeleteProductImage(t.Context(), "p1", saved.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductImageNotFound)
	})

	t.Run("First image is added at position 0", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		img := &application.ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
		saved, err := env.Svc.AddProductImage(t.Context(), "p1", img)
		require.NoError(t, err)
		assert.Equal(t, 0, saved.Position)

		p, err := env.Repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 2, p.Version)
	})

	t.Run("Additional images are appended", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		img := &application.ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
		_, err := env.Svc.AddProductImage(t.Context(), "p1", img)
		require.NoError(t, err)

		img2 := &application.ImageUpload{Reader: bytes.NewReader([]byte("y")), Filename: "b.jpg", ContentType: "image/jpeg", Size: 10}
		saved2, err := env.Svc.AddProductImage(t.Context(), "p1", img2)
		require.NoError(t, err)
		assert.Equal(t, 1, saved2.Position)

		p, err := env.Repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 3, p.Version)
	})

	t.Run("Product response includes its images", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		img := &application.ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
		_, err := env.Svc.AddProductImage(t.Context(), "p1", img)
		require.NoError(t, err)

		images, err := env.Svc.ListProductImages(t.Context(), "p1")
		require.NoError(t, err)
		assert.Len(t, images, 1)
	})

	t.Run("Image is deleted", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		img := &application.ImageUpload{Reader: bytes.NewReader([]byte("x")), Filename: "a.jpg", ContentType: "image/jpeg", Size: 10}
		saved, err := env.Svc.AddProductImage(t.Context(), "p1", img)
		require.NoError(t, err)

		_, err = env.Svc.DeleteProductImage(t.Context(), "p1", saved.ID)
		require.NoError(t, err)

		images, err := env.Svc.ListProductImages(t.Context(), "p1")
		require.NoError(t, err)
		assert.Empty(t, images)

		p, err := env.Repo.GetByID(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 3, p.Version)
	})
}

func TestCatalogRepository_ProductImages(t *testing.T) {
	ctx := t.Context()

	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err = seedTestCategory(ctx, pool); err != nil {
		t.Fatalf("seed test category: %v", err)
	}

	repo := postgres.NewCatalogRepository(pool, pool, outbox.NewPublisher(outboxpg.NewStore(pool)))

	_, _ = pool.Exec(t.Context(), "DELETE FROM product_images WHERE product_id = $1", "it-images")
	_, _ = pool.Exec(t.Context(), "DELETE FROM products WHERE id = $1", "it-images")

	product, err := domain.NewProduct("it-images", "Image Widget", 500, 3, "cat-1", "", "", nil)
	if err != nil {
		t.Fatalf("NewProduct: %v", err)
	}
	if err = repo.Save(ctx, product); err != nil {
		t.Fatalf("Save: %v", err)
	}

	first, err := domain.NewProductImage("it-images", "products/it-images/images/first.jpg", "bucket", "image/jpeg", 10)
	if err != nil {
		t.Fatalf("NewProductImage: %v", err)
	}
	first.ID = "it-image-first"
	first, err = repo.SaveProductImage(ctx, first)
	if err != nil {
		t.Fatalf("SaveProductImage(first): %v", err)
	}
	if first.Position != 0 {
		t.Fatalf("first position = %d, want 0", first.Position)
	}

	second, err := domain.NewProductImage("it-images", "products/it-images/images/second.png", "bucket", "image/png", 20)
	if err != nil {
		t.Fatalf("NewProductImage: %v", err)
	}
	second.ID = "it-image-second"
	second, err = repo.SaveProductImage(ctx, second)
	if err != nil {
		t.Fatalf("SaveProductImage(second): %v", err)
	}
	if second.Position != 1 {
		t.Fatalf("second position = %d, want 1", second.Position)
	}

	stored, err := repo.GetByID(ctx, "it-images")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if stored.Version != 3 {
		t.Fatalf("product version after two images = %d, want 3", stored.Version)
	}

	images, err := repo.ListProductImages(ctx, "it-images")
	if err != nil {
		t.Fatalf("ListProductImages: %v", err)
	}
	if len(images) != 2 || images[0].ID != first.ID || images[1].ID != second.ID {
		t.Fatalf("images out of order: %+v", images)
	}

	grouped, err := repo.ListProductImagesByProductIDs(ctx, []string{"it-images", "it-missing"})
	if err != nil {
		t.Fatalf("ListProductImagesByProductIDs: %v", err)
	}
	if len(grouped["it-images"]) != 2 {
		t.Fatalf("grouped images = %d, want 2", len(grouped["it-images"]))
	}
	if len(grouped["it-missing"]) != 0 {
		t.Fatalf("missing product images = %d, want 0", len(grouped["it-missing"]))
	}

	deleted, err := repo.DeleteProductImage(ctx, "it-images", first.ID)
	if err != nil {
		t.Fatalf("DeleteProductImage: %v", err)
	}
	if deleted.ID != first.ID {
		t.Fatalf("deleted id = %q, want %q", deleted.ID, first.ID)
	}

	stored, err = repo.GetByID(ctx, "it-images")
	if err != nil {
		t.Fatalf("GetByID after delete: %v", err)
	}
	if stored.Version != 4 {
		t.Fatalf("product version after delete = %d, want 4", stored.Version)
	}

	if _, err = repo.DeleteProductImage(ctx, "it-images", first.ID); !errors.Is(err, domain.ErrProductImageNotFound) {
		t.Fatalf("DeleteProductImage(removed): want ErrProductImageNotFound, got %v", err)
	}
	if _, err = repo.SaveProductImage(ctx, domain.ProductImage{ID: "it-image-ghost", ProductID: "it-missing"}); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("SaveProductImage(missing product): want ErrProductNotFound, got %v", err)
	}
}
