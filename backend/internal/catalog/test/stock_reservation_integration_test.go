//go:build integration

package test

import (
	"errors"
	"fmt"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/postgres"
	"minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// uniqueCatalogID returns an id unlikely to collide with another run's rows.
func uniqueCatalogID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func newCatalogRepo(t *testing.T) (*postgres.CatalogRepository, *pgxpool.Pool) {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), testDSN)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	// Seed the test category
	if err := seedTestCategory(t.Context(), pool); err != nil {
		t.Fatalf("seed test category: %v", err)
	}

	return postgres.NewCatalogRepository(pool, pool, outbox.NewPublisher(outboxpg.NewStore(pool))), pool
}

func seedIntegrationProduct(t *testing.T, repo *postgres.CatalogRepository, id string, stock int) {
	t.Helper()
	p, err := domain.NewProduct(id, "Widget", 500, stock, "cat-1", "", "", nil)
	if err != nil {
		t.Fatalf("NewProduct: %v", err)
	}
	if err := repo.Save(t.Context(), p); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func stockOf(t *testing.T, repo *postgres.CatalogRepository, id string) int {
	t.Helper()
	p, err := repo.GetByID(t.Context(), id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	return p.Stock
}

func TestCatalogRepository_ReserveStock(t *testing.T) {
	ctx := t.Context()
	repo, _ := newCatalogRepo(t)
	id := uniqueCatalogID("it-reserve")
	seedIntegrationProduct(t, repo, id, 5)

	if err := repo.ReserveStock(ctx, "o1", []domain.ReservationItem{{ProductID: id, Quantity: 2}}); err != nil {
		t.Fatalf("ReserveStock: %v", err)
	}
	if got := stockOf(t, repo, id); got != 3 {
		t.Fatalf("stock after reserve = %d, want 3", got)
	}

	// Re-reserving the same order is a no-op: the reservation already exists.
	if err := repo.ReserveStock(ctx, "o1", []domain.ReservationItem{{ProductID: id, Quantity: 2}}); err != nil {
		t.Fatalf("ReserveStock (retry): %v", err)
	}
	if got := stockOf(t, repo, id); got != 3 {
		t.Fatalf("stock after retry = %d, want 3 (no double reserve)", got)
	}

	// A shortage rejects the whole reservation and leaves stock untouched.
	err := repo.ReserveStock(ctx, "o2", []domain.ReservationItem{{ProductID: id, Quantity: 4}})
	if !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("ReserveStock(short): want ErrInsufficientStock, got %v", err)
	}
	if got := stockOf(t, repo, id); got != 3 {
		t.Fatalf("stock after failed reserve = %d, want 3", got)
	}

	// An unknown product reports the domain sentinel.
	err = repo.ReserveStock(ctx, "o3", []domain.ReservationItem{{ProductID: "it-missing-" + id, Quantity: 1}})
	if !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("ReserveStock(missing): want ErrProductNotFound, got %v", err)
	}
}

func TestCatalogRepository_ReleaseStock(t *testing.T) {
	ctx := t.Context()
	repo, _ := newCatalogRepo(t)
	id := uniqueCatalogID("it-release")
	seedIntegrationProduct(t, repo, id, 5)

	if err := repo.ReserveStock(ctx, "o1", []domain.ReservationItem{{ProductID: id, Quantity: 2}}); err != nil {
		t.Fatalf("ReserveStock: %v", err)
	}
	if err := repo.ReleaseStock(ctx, "o1"); err != nil {
		t.Fatalf("ReleaseStock: %v", err)
	}
	if got := stockOf(t, repo, id); got != 5 {
		t.Fatalf("stock after release = %d, want 5", got)
	}

	// A redelivered release must not add stock twice.
	if err := repo.ReleaseStock(ctx, "o1"); err != nil {
		t.Fatalf("ReleaseStock (retry): %v", err)
	}
	if got := stockOf(t, repo, id); got != 5 {
		t.Fatalf("stock after retried release = %d, want 5", got)
	}
}

// TestCatalogRepository_ReserveStockConcurrent proves the conditional UPDATE
// prevents overselling: with one unit in stock, two racing reservations for
// different orders yield exactly one success and one shortage.
func TestCatalogRepository_ReserveStockConcurrent(t *testing.T) {
	repo, _ := newCatalogRepo(t)
	id := uniqueCatalogID("it-race")
	seedIntegrationProduct(t, repo, id, 1)

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, orderID := range []string{"o1", "o2"} {
		wg.Add(1)
		go func(orderID string) {
			defer wg.Done()
			errs <- repo.ReserveStock(t.Context(), orderID, []domain.ReservationItem{{ProductID: id, Quantity: 1}})
		}(orderID)
	}
	wg.Wait()
	close(errs)

	var success, shortage int
	for err := range errs {
		switch {
		case err == nil:
			success++
		case errors.Is(err, domain.ErrInsufficientStock):
			shortage++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if success != 1 || shortage != 1 {
		t.Fatalf("want 1 success and 1 shortage, got success=%d shortage=%d", success, shortage)
	}
	if got := stockOf(t, repo, id); got != 0 {
		t.Fatalf("stock after race = %d, want 0", got)
	}
}
