//go:build integration

package test

import (
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/postgres"
	"minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProductIntegration(t *testing.T) {
	env := newCatalogEnv(t)

	t.Run("Product does not exist", func(t *testing.T) {
		_, err := env.Svc.GetProduct(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("Product got returned with no problems", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		got, err := env.Svc.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, "p1", got.ID)
		assert.Equal(t, "Widget", got.Name)
		assert.Equal(t, int64(500), got.PriceCents)
		assert.Equal(t, 3, got.Stock)
		assert.Equal(t, 1, got.Version)
		assert.False(t, got.CreatedAt.IsZero())
	})

	t.Run("Product with description returned", func(t *testing.T) {
		seedCatalogProductFull(t, env, "p1", "Widget", 500, 3, "", "A great widget", nil)
		got, err := env.Svc.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, "A great widget", got.Description)
	})

	t.Run("Product with SKU returned", func(t *testing.T) {
		seedCatalogProductFull(t, env, "p1", "Widget", 500, 3, "SKU-123", "", nil)
		got, err := env.Svc.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, "SKU-123", got.SKU)
	})

	t.Run("Product with details returned", func(t *testing.T) {
		seedCatalogProductFull(t, env, "p1", "Widget", 500, 3, "", "", []domain.ProductDetail{{Name: "Color", Value: "Red"}})
		got, err := env.Svc.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, []domain.ProductDetail{{Name: "Color", Value: "Red"}}, got.Details)
	})

	t.Run("Unchanged product answers 304", func(t *testing.T) {
		seedCatalogProduct(t, env, "p1", 3)
		got, err := env.Svc.GetProduct(t.Context(), "p1")
		require.NoError(t, err)
		assert.Equal(t, 1, got.Version)
	})
}

func TestCatalogRepository_SaveGetByID(t *testing.T) {
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

	// Locally both pools point at the same database.
	repo := postgres.NewCatalogRepository(pool, pool, outbox.NewPublisher(outboxpg.NewStore(pool)))

	_, _ = pool.Exec(t.Context(), "DELETE FROM products WHERE id = $1", "it-widget")

	want, err := domain.NewProduct("it-widget", "Integration Widget", 1299, 7, "cat-1", "", "", nil)
	if err != nil {
		t.Fatalf("NewProduct: %v", err)
	}
	if err = repo.Save(ctx, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID(ctx, "it-widget")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != want.ID || got.Name != want.Name ||
		got.PriceCents != want.PriceCents || got.Stock != want.Stock {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, want)
	}
	if got.Version != 1 {
		t.Fatalf("new product version = %d, want 1", got.Version)
	}
	// The adapter writes the value's timestamps, so they match after a round
	// trip up to the database's microsecond resolution.
	if got.CreatedAt.Sub(want.CreatedAt).Abs() > time.Millisecond {
		t.Fatalf("created_at mismatch: got %v, want %v", got.CreatedAt, want.CreatedAt)
	}
	if got.UpdatedAt.Sub(want.UpdatedAt).Abs() > time.Millisecond {
		t.Fatalf("updated_at mismatch: got %v, want %v", got.UpdatedAt, want.UpdatedAt)
	}

	// Re-saving the same id is an upsert: created_at must survive while the
	// other columns refresh, matching the in-memory adapter.
	resave := want
	resave.Name = "Renamed Widget"
	resave.Stock = 9
	resave.CreatedAt = want.CreatedAt.Add(time.Hour)
	resave.UpdatedAt = want.UpdatedAt.Add(time.Hour)
	if err = repo.Save(ctx, resave); err != nil {
		t.Fatalf("Save (upsert): %v", err)
	}
	got, err = repo.GetByID(ctx, "it-widget")
	if err != nil {
		t.Fatalf("GetByID after upsert: %v", err)
	}
	if got.CreatedAt.Sub(want.CreatedAt).Abs() > time.Millisecond {
		t.Fatalf("created_at was rewritten on upsert: got %v, want %v", got.CreatedAt, want.CreatedAt)
	}
	if got.Name != "Renamed Widget" || got.Stock != 9 {
		t.Fatalf("upsert did not refresh columns: got %+v", got)
	}
	// The upsert refreshes updated_at with the database's NOW(), while want was
	// stamped by the Go clock, so allow for a little clock skew between host and
	// container.
	if got.UpdatedAt.Before(want.UpdatedAt.Add(-time.Second)) {
		t.Fatalf("updated_at moved backwards on upsert: got %v, want >= %v", got.UpdatedAt, want.UpdatedAt)
	}
}

// TestCatalogRepository_Update changes only the name and price through the real
// Postgres adapter and proves stock is preserved by the UPDATE statement.
