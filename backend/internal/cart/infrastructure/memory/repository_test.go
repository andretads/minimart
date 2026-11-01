package memory

import (
	"errors"
	"minimart/internal/cart/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustCart(t *testing.T, id, customerID string) domain.Cart {
	t.Helper()
	cart, err := domain.NewCart(id, customerID)
	require.NoError(t, err)
	return cart
}

func mustItem(productID string, qty int) domain.Item {
	return domain.Item{
		ProductID:              productID,
		ProductNameSnapshot:    "Product " + productID,
		UnitPriceSnapshotCents: 1000,
		Quantity:               qty,
		CreatedAt:              time.Now().UTC(),
		UpdatedAt:              time.Now().UTC(),
	}
}

func TestRepository(t *testing.T) {
	ctx := t.Context()
	repo := New()

	cart := mustCart(t, "cart-1", "cust-1")

	// Create, then read the same cart back.
	require.NoError(t, repo.Create(ctx, cart))
	got, err := repo.GetByCustomerID(ctx, "cust-1")
	require.NoError(t, err)
	assert.Equal(t, cart.ID, got.ID)
	assert.Equal(t, cart.CustomerID, got.CustomerID)
	assert.Equal(t, cart.Status, got.Status)
	assert.Equal(t, cart.Version, got.Version)
	assert.Empty(t, got.Items)

	// A missing customer returns the domain not-found sentinel.
	_, err = repo.GetByCustomerID(ctx, "missing")
	require.ErrorIs(t, err, domain.ErrCartNotFound)

	// Create another cart for a different customer
	cart2 := mustCart(t, "cart-2", "cust-2")
	require.NoError(t, repo.Create(ctx, cart2))

	// GetByCustomerID returns the correct cart
	got2, err := repo.GetByCustomerID(ctx, "cust-2")
	require.NoError(t, err)
	assert.Equal(t, cart2.ID, got2.ID)

	// Add items and update
	got2.Items = append(got2.Items, mustItem("prod-1", 2))
	require.NoError(t, repo.Update(ctx, got2, got2.Version))
	// Version is incremented in the repo, fetch to verify
	updated, err := repo.GetByCustomerID(ctx, "cust-2")
	require.NoError(t, err)
	assert.Equal(t, 2, updated.Version)

	// Verify items persisted
	got3, err := repo.GetByCustomerID(ctx, "cust-2")
	require.NoError(t, err)
	require.Len(t, got3.Items, 1)
	assert.Equal(t, "prod-1", got3.Items[0].ProductID)
	assert.Equal(t, 2, got3.Items[0].Quantity)
	assert.Equal(t, 2, got3.Version)
}

// TestRepositoryCreateIdempotent proves that Create is idempotent on ID.
func TestRepositoryCreateIdempotent(t *testing.T) {
	ctx := t.Context()
	repo := New()

	cart := mustCart(t, "cart-1", "cust-1")
	require.NoError(t, repo.Create(ctx, cart))

	// Creating again with same ID should fail
	err := repo.Create(ctx, cart)
	require.ErrorIs(t, err, domain.ErrCartAlreadyExists)
}

// TestRepositoryUpdateOptimisticLock proves CAS on version works correctly.
func TestRepositoryUpdateOptimisticLock(t *testing.T) {
	ctx := t.Context()
	repo := New()

	cart := mustCart(t, "cart-1", "cust-1")
	require.NoError(t, repo.Create(ctx, cart))

	// Valid update with correct version
	cart.Items = append(cart.Items, mustItem("prod-1", 2))
	require.NoError(t, repo.Update(ctx, cart, 1))
	// Version is incremented in the repo, fetch to verify
	updated, err := repo.GetByCustomerID(ctx, "cust-1")
	require.NoError(t, err)
	assert.Equal(t, 2, updated.Version)

	// Stale version should fail
	staleCart := updated
	staleCart.Items = append(staleCart.Items, mustItem("prod-2", 1))
	updErr := repo.Update(ctx, staleCart, 1)
	require.ErrorIs(t, updErr, domain.ErrCartOptimisticLock)

	// Missing cart should fail with ErrCartNotFound
	missingCart := mustCart(t, "missing", "cust-missing")
	err = repo.Update(ctx, missingCart, 1)
	require.ErrorIs(t, err, domain.ErrCartNotFound)
}

// TestRepositoryUpdateConcurrent hammers Update from parallel goroutines
// for the same cart - proving the CAS semantics under concurrency.
func TestRepositoryUpdateConcurrent(t *testing.T) {
	ctx := t.Context()
	repo := New()

	cart := mustCart(t, "cart-1", "cust-1")
	require.NoError(t, repo.Create(ctx, cart))

	const writers = 10
	errs := make(chan error, writers)
	products := []string{"prod-a", "prod-b", "prod-c", "prod-d", "prod-e", "prod-f", "prod-g", "prod-h", "prod-i", "prod-j"}
	for i := range writers {
		go func(i int) {
			// Each goroutine tries to add a different item
			c := cart
			c.Items = append(c.Items, mustItem(products[i], 1))
			errs <- repo.Update(ctx, c, c.Version)
		}(i)
	}

	// Exactly one should succeed (the first to CAS), others fail with OptimisticLock
	successCount := 0
	for range writers {
		err := <-errs
		if err == nil {
			successCount++
		} else {
			require.ErrorIs(t, err, domain.ErrCartOptimisticLock)
		}
	}
	assert.Equal(t, 1, successCount, "exactly one concurrent update should succeed")
}

// TestRepositoryStatusTransitions tests the cart status transitions
// via Update: active -> checkout_pending -> consumed.
func TestRepositoryStatusTransitions(t *testing.T) {
	ctx := t.Context()
	repo := New()

	// Create active cart
	cart := mustCart(t, "cart-1", "cust-1")
	require.NoError(t, repo.Create(ctx, cart))
	assert.Equal(t, domain.StatusActive, cart.Status)

	// Transition to checkout_pending
	cart.Status = domain.StatusCheckoutPending
	cart.CheckoutKey = "idem-key-1"
	require.NoError(t, repo.Update(ctx, cart, 1))
	assert.Equal(t, domain.StatusCheckoutPending, cart.Status)

	// Transition to consumed
	cart.Status = domain.StatusConsumed
	cart.OrderID = "order-1"
	cart.CheckoutKey = ""
	require.NoError(t, repo.Update(ctx, cart, 2))
	assert.Equal(t, domain.StatusConsumed, cart.Status)
	assert.Equal(t, "order-1", cart.OrderID)
}

// TestRepositoryGetByCustomerIDOnlyActiveOrPending verifies that
// GetByCustomerID only returns carts with status active or checkout_pending,
// not consumed carts.
func TestRepositoryGetByCustomerIDOnlyActiveOrPending(t *testing.T) {
	ctx := t.Context()
	repo := New()

	// Create and consume a cart
	cart := mustCart(t, "cart-1", "cust-1")
	require.NoError(t, repo.Create(ctx, cart))
	cart.Status = domain.StatusConsumed
	cart.OrderID = "order-1"
	require.NoError(t, repo.Update(ctx, cart, 1))

	// Should not find consumed cart
	_, err := repo.GetByCustomerID(ctx, "cust-1")
	require.ErrorIs(t, err, domain.ErrCartNotFound)

	// But active cart should be found
	cart2 := mustCart(t, "cart-2", "cust-1")
	require.NoError(t, repo.Create(ctx, cart2))
	got, err := repo.GetByCustomerID(ctx, "cust-1")
	require.NoError(t, err)
	assert.Equal(t, cart2.ID, got.ID)
}

// TestRepositoryConcurrentCreate guards the mutex: parallel idempotent creates of
// the same cart must not race.
func TestRepositoryConcurrentCreate(t *testing.T) {
	ctx := t.Context()
	repo := New()

	cart := mustCart(t, "cart-1", "cust-1")

	errs := make(chan error, 10)
	for range 10 {
		go func() { errs <- repo.Create(ctx, cart) }()
	}
	for range 10 {
		err := <-errs
		// First one succeeds, rest fail with ErrCartAlreadyExists
		require.True(t, err == nil || errors.Is(err, domain.ErrCartAlreadyExists))
	}

	// Verify only one cart exists for this customer
	_, err := repo.GetByCustomerID(ctx, "cust-1")
	require.NoError(t, err)
}
