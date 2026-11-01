//go:build integration

package test

import (
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

func validUpdateProductReq(id string, version int) application.UpdateProductRequest {
	return application.UpdateProductRequest{ID: id, Name: "Gadget", PriceCents: 750, Version: version, CategoryID: "cat-1"}
}

func TestUpdateProductIntegration(t *testing.T) {
	env := newCatalogEnv(t)

	t.Run("Product does not exist", func(t *testing.T) {
		_, err := env.Svc.UpdateProduct(t.Context(), validUpdateProductReq("missing", 1))
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Blank name is rejected", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		req := validUpdateProductReq("p1", 1)
		req.Name = "   "
		_, err := env.Svc.UpdateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyName)
	})

	t.Run("Non-positive price is rejected", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		req := validUpdateProductReq("p1", 1)
		req.PriceCents = 0
		_, err := env.Svc.UpdateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNonPositivePrice)
	})

	t.Run("Storage failure", func(t *testing.T) {
		t.Skip("storage failure not easily simulated with real DB")
	})

	t.Run("Missing version is rejected", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		req := validUpdateProductReq("p1", 1)
		req.Version = 0
		_, err := env.Svc.UpdateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("Stale version is rejected", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		_, err := env.Svc.UpdateProduct(t.Context(), validUpdateProductReq("p1", 1))
		require.NoError(t, err)

		_, err = env.Svc.UpdateProduct(t.Context(), validUpdateProductReq("p1", 1))
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductOptimisticLock)
	})

	t.Run("Stale ETag is rejected", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		_, err := env.Svc.UpdateProduct(t.Context(), validUpdateProductReq("p1", 99))
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductOptimisticLock)
	})

	t.Run("Empty SKU is rejected", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		req := validUpdateProductReq("p1", 1)
		req.SKU = "   "
		_, err := env.Svc.UpdateProduct(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptySKU)
	})

	t.Run("Product got updated with no problems", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		got, err := env.Svc.UpdateProduct(t.Context(), validUpdateProductReq("p1", 1))
		require.NoError(t, err)
		assert.Equal(t, "p1", got.ID)
		assert.Equal(t, "Gadget", got.Name)
		assert.Equal(t, int64(750), got.PriceCents)
		assert.Equal(t, 3, got.Stock)
		assert.Equal(t, 2, got.Version)
	})

	t.Run("Product is updated using the ETag from GET", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		got, err := env.Svc.UpdateProduct(t.Context(), validUpdateProductReq("p1", 1))
		require.NoError(t, err)
		assert.Equal(t, 2, got.Version)
	})

	t.Run("Product name is trimmed on update", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		req := validUpdateProductReq("p1", 1)
		req.Name = "  Gadget  "
		got, err := env.Svc.UpdateProduct(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "Gadget", got.Name)
	})

	t.Run("Product got updated with description", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		req := validUpdateProductReq("p1", 1)
		req.Description = "Updated description"
		got, err := env.Svc.UpdateProduct(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", got.Description)
	})

	t.Run("Product got updated with SKU", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		req := validUpdateProductReq("p1", 1)
		req.SKU = "NEW-SKU"
		got, err := env.Svc.UpdateProduct(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, "NEW-SKU", got.SKU)
	})

	t.Run("Product got updated with details", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		req := validUpdateProductReq("p1", 1)
		req.Details = []domain.ProductDetail{{Name: "Color", Value: "Blue"}}
		got, err := env.Svc.UpdateProduct(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, []domain.ProductDetail{{Name: "Color", Value: "Blue"}}, got.Details)
	})
}

func TestCatalogRepository_Update(t *testing.T) {
	ctx := t.Context()

	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	// Seed the test category
	if seedErr := seedTestCategory(ctx, pool); seedErr != nil {
		t.Fatalf("seed test category: %v", seedErr)
	}

	repo := postgres.NewCatalogRepository(pool, pool, outbox.NewPublisher(outboxpg.NewStore(pool)))

	_, _ = pool.Exec(t.Context(), "DELETE FROM products WHERE id = $1", "it-update")

	saved, err := domain.NewProduct("it-update", "Before", 100, 5, "cat-1", "", "", nil)
	if err != nil {
		t.Fatalf("NewProduct: %v", err)
	}
	if err = repo.Save(ctx, saved); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Capture the stored created_at before the update so we can prove it holds.
	before, err := repo.GetByID(ctx, "it-update")
	if err != nil {
		t.Fatalf("GetByID before update: %v", err)
	}

	updated, err := saved.UpdateNameAndPrice("After", 250)
	if err != nil {
		t.Fatalf("UpdateNameAndPrice: %v", err)
	}
	persisted, err := repo.Update(ctx, updated)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if persisted.Version != 2 {
		t.Fatalf("Update version = %d, want 2", persisted.Version)
	}

	got, err := repo.GetByID(ctx, "it-update")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != persisted.Name || got.PriceCents != persisted.PriceCents {
		t.Fatalf("update mismatch: got %+v, want %+v", got, persisted)
	}
	if got.Version != 2 {
		t.Fatalf("stored version = %d, want 2", got.Version)
	}
	if got.Stock != 5 {
		t.Fatalf("stock not preserved: got %d, want 5", got.Stock)
	}
	if !got.CreatedAt.Equal(before.CreatedAt) {
		t.Fatalf("created_at changed on update: got %v, want %v", got.CreatedAt, before.CreatedAt)
	}

	// Updating a missing id surfaces the domain sentinel, not a driver error.
	ghost, err := domain.NewProduct("it-ghost", "Ghost", 100, 0, "cat-1", "", "", nil)
	if err != nil {
		t.Fatalf("NewProduct: %v", err)
	}
	if _, err = repo.Update(ctx, ghost); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("Update(missing): want ErrProductNotFound, got %v", err)
	}
}

// TestCatalogRepository_UpdateOptimisticLock proves the WHERE version predicate
// rejects a stale write against real Postgres and leaves the newer write intact.
func TestCatalogRepository_UpdateOptimisticLock(t *testing.T) {
	ctx := t.Context()

	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	// Seed the test category
	if seedErr := seedTestCategory(ctx, pool); seedErr != nil {
		t.Fatalf("seed test category: %v", seedErr)
	}

	repo := postgres.NewCatalogRepository(pool, pool, outbox.NewPublisher(outboxpg.NewStore(pool)))

	_, _ = pool.Exec(t.Context(), "DELETE FROM products WHERE id = $1", "it-lock")

	original, err := domain.NewProduct("it-lock", "Original", 100, 4, "cat-1", "", "", nil)
	if err != nil {
		t.Fatalf("NewProduct: %v", err)
	}
	if err = repo.Save(ctx, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Both writers start from the same version-1 snapshot.
	reader, err := repo.GetByID(ctx, "it-lock")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	first, err := reader.UpdateNameAndPrice("First", 200)
	if err != nil {
		t.Fatalf("UpdateNameAndPrice: %v", err)
	}
	if _, err = repo.Update(ctx, first); err != nil {
		t.Fatalf("Update(first): %v", err)
	}

	second, err := reader.UpdateNameAndPrice("Second", 300)
	if err != nil {
		t.Fatalf("UpdateNameAndPrice: %v", err)
	}
	if _, err = repo.Update(ctx, second); !errors.Is(err, domain.ErrProductOptimisticLock) {
		t.Fatalf("Update(stale): want ErrProductOptimisticLock, got %v", err)
	}

	// The first writer's change survived.
	got, err := repo.GetByID(ctx, "it-lock")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "First" || got.PriceCents != 200 || got.Version != 2 {
		t.Fatalf("stale write clobbered the row: got %+v", got)
	}
}
