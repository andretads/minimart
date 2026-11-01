//go:build integration

package test

import (
	"errors"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/postgres"
	"minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
