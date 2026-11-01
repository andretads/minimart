//go:build integration

// Package test holds the catalog domain's integration tests. Unlike the unit
// tests, these run against a REAL Postgres and prove the SQL wiring the unit
// tests deliberately fake out.
//
// The database is provided by testcontainers-go: TestMain (main_test.go) starts
// an ephemeral Postgres, applies the repo's db/migrations, and tears it down
// afterwards. No docker-compose or manual `go run . migrate` is required. They
// are excluded from a normal `go test ./...` by the build tag above; run them
// with:
//
//	go test -count=1 -tags=integration ./internal/catalog/test/
package test

import (
	"context"
	"errors"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/postgres"
	"minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedTestCategory creates the test category in the database if it doesn't exist.
func seedTestCategory(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO categories (id, name, icon, version, created_at, updated_at)
		VALUES ('cat-1', 'Test Category', '', 1, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`)
	return err
}

// TestCatalogRepository_SaveGetByID round-trips a product through the real Postgres
// adapter: Save, then GetByID, and asserts the value comes back unchanged. The
// upsert query makes the fixed ID safe to re-run.
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
