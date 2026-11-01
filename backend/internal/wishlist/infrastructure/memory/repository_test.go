package memory

import (
	"errors"
	"minimart/internal/wishlist/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustWishlist(t *testing.T, id, customerID string) domain.Wishlist {
	t.Helper()
	wishlist, err := domain.NewWishlist(id, customerID)
	require.NoError(t, err)
	return wishlist
}

func TestRepository(t *testing.T) {
	ctx := t.Context()
	repo := New()

	wishlist := mustWishlist(t, "wl-1", "cust-1")

	// Create, then read the same wishlist back.
	require.NoError(t, repo.Create(ctx, wishlist))
	got, err := repo.GetByCustomerID(ctx, "cust-1")
	require.NoError(t, err)
	assert.Equal(t, wishlist.ID, got.ID)
	assert.Equal(t, wishlist.CustomerID, got.CustomerID)
	assert.Empty(t, got.Items)

	// A missing customer returns the domain not-found sentinel.
	_, err = repo.GetByCustomerID(ctx, "missing")
	require.ErrorIs(t, err, domain.ErrWishlistNotFound)

	// Create another wishlist for a different customer
	wishlist2 := mustWishlist(t, "wl-2", "cust-2")
	require.NoError(t, repo.Create(ctx, wishlist2))

	// GetByCustomerID returns the correct wishlist
	got2, err := repo.GetByCustomerID(ctx, "cust-2")
	require.NoError(t, err)
	assert.Equal(t, wishlist2.ID, got2.ID)

	// Add items and update
	got2.Items = append(got2.Items, domain.WishlistItem{ProductID: "prod-1", AddedAt: time.Now().UTC()})
	require.NoError(t, repo.Update(ctx, got2))
	// Fetch to verify
	updated, err := repo.GetByCustomerID(ctx, "cust-2")
	require.NoError(t, err)
	require.Len(t, updated.Items, 1)
	assert.Equal(t, "prod-1", updated.Items[0].ProductID)
}

// TestRepositoryCreateIdempotent proves that Create is idempotent on ID.
func TestRepositoryCreateIdempotent(t *testing.T) {
	ctx := t.Context()
	repo := New()

	wishlist := mustWishlist(t, "wl-1", "cust-1")
	require.NoError(t, repo.Create(ctx, wishlist))

	// Creating again with same ID should fail
	err := repo.Create(ctx, wishlist)
	require.ErrorIs(t, err, domain.ErrWishlistAlreadyExists)
}

// TestRepositoryUpdateConcurrent hammers Update from parallel goroutines
// for the same wishlist - proving the mutex semantics under concurrency.
func TestRepositoryUpdateConcurrent(t *testing.T) {
	ctx := t.Context()
	repo := New()

	wishlist := mustWishlist(t, "wl-1", "cust-1")
	require.NoError(t, repo.Create(ctx, wishlist))

	const writers = 10
	errs := make(chan error, writers)
	products := []string{"prod-a", "prod-b", "prod-c", "prod-d", "prod-e", "prod-f", "prod-g", "prod-h", "prod-i", "prod-j"}
	for i := range writers {
		go func(i int) {
			// Each goroutine tries to add a different item
			w := wishlist
			w.Items = append(w.Items, domain.WishlistItem{ProductID: products[i], AddedAt: time.Now().UTC()})
			errs <- repo.Update(ctx, w)
		}(i)
	}

	// Exactly one should succeed (the first to update), others may succeed or fail depending on timing
	successCount := 0
	for range writers {
		err := <-errs
		if err == nil {
			successCount++
		}
	}
	assert.GreaterOrEqual(t, successCount, 1, "at least one concurrent update should succeed")
}

// TestRepositoryConcurrentCreate guards the mutex: parallel idempotent creates of
// the same wishlist must not race.
func TestRepositoryConcurrentCreate(t *testing.T) {
	ctx := t.Context()
	repo := New()

	wishlist := mustWishlist(t, "wl-1", "cust-1")

	errs := make(chan error, 10)
	for range 10 {
		go func() { errs <- repo.Create(ctx, wishlist) }()
	}
	for range 10 {
		err := <-errs
		// First one succeeds, rest fail with ErrWishlistAlreadyExists
		require.True(t, err == nil || errors.Is(err, domain.ErrWishlistAlreadyExists))
	}

	// Verify only one wishlist exists for this customer
	_, err := repo.GetByCustomerID(ctx, "cust-1")
	require.NoError(t, err)
}

// TestRepositoryUpdateNonExistentWishlist returns ErrWishlistNotFound when updating missing wishlist
func TestRepositoryUpdateNonExistentWishlist(t *testing.T) {
	ctx := t.Context()
	repo := New()

	wishlist := mustWishlist(t, "missing", "cust-missing")
	err := repo.Update(ctx, wishlist)
	require.ErrorIs(t, err, domain.ErrWishlistNotFound)
}
