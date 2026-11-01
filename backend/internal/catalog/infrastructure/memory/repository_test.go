package memory

import (
	"errors"
	"minimart/internal/catalog/domain"
	"testing"
)

func TestRepository(t *testing.T) {
	ctx := t.Context()
	repo := New()

	p, err := domain.NewProduct("p-1", "Widget", 500, 3, "cat-1", "", "", nil)
	if err != nil {
		t.Fatalf("NewProduct: %v", err)
	}

	// Save, then read the same product back.
	if err = repo.Save(ctx, p); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.GetByID(ctx, "p-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != p.ID || got.Name != p.Name || got.PriceCents != p.PriceCents || got.Stock != p.Stock || got.Version != p.Version || got.CategoryID != p.CategoryID {
		t.Fatalf("GetByID returned %+v, want %+v", got, p)
	}

	// List returns the one saved product.
	all, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("List returned %d products, want 1", len(all))
	}

	// A missing id returns the domain not-found sentinel.
	if _, err = repo.GetByID(ctx, "missing"); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("GetByID(missing): want ErrProductNotFound, got %v", err)
	}

	// Update applies the change, bumps the version and returns the stored product.
	updated, err := p.UpdateNameAndPrice("Gadget", 750)
	if err != nil {
		t.Fatalf("UpdateNameAndPrice: %v", err)
	}
	persisted, err := repo.Update(ctx, updated)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if persisted.Version != 2 {
		t.Fatalf("Update bumped version to %d, want 2", persisted.Version)
	}
	if !persisted.CreatedAt.Equal(p.CreatedAt) {
		t.Fatalf("Update changed created_at: got %v, want %v", persisted.CreatedAt, p.CreatedAt)
	}
	got, err = repo.GetByID(ctx, "p-1")
	if err != nil {
		t.Fatalf("GetByID after Update: %v", err)
	}
	if got.ID != persisted.ID || got.Name != persisted.Name || got.PriceCents != persisted.PriceCents || got.Stock != persisted.Stock || got.Version != persisted.Version || got.CategoryID != persisted.CategoryID {
		t.Fatalf("GetByID after Update returned %+v, want %+v", got, persisted)
	}

	// Updating an unknown id returns the not-found sentinel.
	missing, err := domain.NewProduct("ghost", "Ghost", 100, 0, "cat-1", "", "", nil)
	if err != nil {
		t.Fatalf("NewProduct: %v", err)
	}
	if _, err := repo.Update(ctx, missing); !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("Update(missing): want ErrProductNotFound, got %v", err)
	}
}

// TestRepositoryOptimisticLock proves a write carrying a stale version is
// rejected instead of silently clobbering the newer value.
func TestRepositoryOptimisticLock(t *testing.T) {
	ctx := t.Context()
	repo := New()

	original, err := domain.NewProduct("p-1", "Widget", 500, 3, "cat-1", "", "", nil)
	if err != nil {
		t.Fatalf("NewProduct: %v", err)
	}
	if err = repo.Save(ctx, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// A first writer reads version 1 and commits; the stored version becomes 2.
	stale, err := repo.GetByID(ctx, "p-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	first, err := stale.UpdateNameAndPrice("First", 600)
	if err != nil {
		t.Fatalf("UpdateNameAndPrice: %v", err)
	}
	if _, err = repo.Update(ctx, first); err != nil {
		t.Fatalf("Update(first): %v", err)
	}

	// A second writer still holds version 1 and must be rejected.
	second, err := stale.UpdateNameAndPrice("Second", 700)
	if err != nil {
		t.Fatalf("UpdateNameAndPrice: %v", err)
	}
	if _, err = repo.Update(ctx, second); !errors.Is(err, domain.ErrProductOptimisticLock) {
		t.Fatalf("Update(stale): want ErrProductOptimisticLock, got %v", err)
	}

	// The first writer's change is intact.
	got, err := repo.GetByID(ctx, "p-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "First" || got.PriceCents != 600 || got.Version != 2 {
		t.Fatalf("stored product was clobbered: got %+v", got)
	}
}
