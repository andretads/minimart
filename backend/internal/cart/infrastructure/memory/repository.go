// Package memory is an in-memory adapter for the cart Repository port.
// Data lives only for the life of the process. It is the simplest possible
// adapter and stays useful forever as a fast, dependency-free double in tests.
package memory

import (
	"context"
	"minimart/internal/cart/domain"
	"sync"
)

// Repository stores carts in a map guarded by a mutex, safe for concurrent use.
type Repository struct {
	mu   sync.RWMutex
	byID map[string]domain.Cart
}

// Compile-time proof that Repository satisfies the port. If a method drifts,
// the build fails here, not at runtime.
var _ domain.Repository = (*Repository)(nil)

func New() *Repository {
	return &Repository{
		byID: make(map[string]domain.Cart),
	}
}

// GetByCustomerID returns the customer's active or checkout_pending cart,
// or domain.ErrCartNotFound.
func (r *Repository) GetByCustomerID(ctx context.Context, customerID string) (domain.Cart, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, cart := range r.byID {
		if cart.CustomerID == customerID && (cart.Status == domain.StatusActive || cart.Status == domain.StatusCheckoutPending) {
			return cart, nil
		}
	}
	return domain.Cart{}, domain.ErrCartNotFound
}

// GetByCustomerIDIncludingConsumed returns the customer's cart regardless of status
// (active, checkout_pending, or consumed), or domain.ErrCartNotFound.
func (r *Repository) GetByCustomerIDIncludingConsumed(ctx context.Context, customerID string) (domain.Cart, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, cart := range r.byID {
		if cart.CustomerID == customerID {
			return cart, nil
		}
	}
	return domain.Cart{}, domain.ErrCartNotFound
}

// Create records the cart. The caller is responsible for ensuring at most one
// active cart per customer (the MongoDB adapter enforces this via a unique
// partial index). If a cart with the same ID already exists, returns
// domain.ErrCartAlreadyExists.
func (r *Repository) Create(ctx context.Context, cart domain.Cart) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byID[cart.ID]; exists {
		return domain.ErrCartAlreadyExists
	}
	r.byID[cart.ID] = cart
	return nil
}

// Update performs an optimistic-lock CAS on (id, version). It sets the mutable
// fields and increments version. A stale version returns
// domain.ErrCartOptimisticLock; a missing cart returns domain.ErrCartNotFound.
func (r *Repository) Update(ctx context.Context, cart domain.Cart, expectedVersion int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.byID[cart.ID]
	if !ok {
		return domain.ErrCartNotFound
	}
	if existing.Version != expectedVersion {
		return domain.ErrCartOptimisticLock
	}
	cart.Version = expectedVersion + 1
	r.byID[cart.ID] = cart
	return nil
}
