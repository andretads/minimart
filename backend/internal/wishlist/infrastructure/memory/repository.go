// Package memory is an in-memory adapter for the wishlist Repository port.
// Data lives only for the life of the process. It is the simplest possible
// adapter and stays useful forever as a fast, dependency-free double in tests.
package memory

import (
	"context"
	"minimart/internal/wishlist/domain"
	"sync"
)

// Repository stores wishlists in a map guarded by a mutex, safe for concurrent use.
type Repository struct {
	mu       sync.RWMutex
	byID     map[string]domain.Wishlist
	byCustID map[string]string // customerID -> wishlistID
}

// Compile-time proof that Repository satisfies the port. If a method drifts,
// the build fails here, not at runtime.
var _ domain.Repository = (*Repository)(nil)

func New() *Repository {
	return &Repository{
		byID:     make(map[string]domain.Wishlist),
		byCustID: make(map[string]string),
	}
}

// GetByCustomerID returns the customer's wishlist, or domain.ErrWishlistNotFound.
func (r *Repository) GetByCustomerID(ctx context.Context, customerID string) (domain.Wishlist, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	wishlistID, ok := r.byCustID[customerID]
	if !ok {
		return domain.Wishlist{}, domain.ErrWishlistNotFound
	}
	wishlist, ok := r.byID[wishlistID]
	if !ok {
		return domain.Wishlist{}, domain.ErrWishlistNotFound
	}
	return wishlist, nil
}

// Create records the wishlist. The caller is responsible for ensuring at most one
// wishlist per customer (the MongoDB adapter enforces this via a unique
// index). If a wishlist with the same ID already exists, returns
// domain.ErrWishlistAlreadyExists.
func (r *Repository) Create(ctx context.Context, wishlist domain.Wishlist) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byID[wishlist.ID]; exists {
		return domain.ErrWishlistAlreadyExists
	}
	if _, exists := r.byCustID[wishlist.CustomerID]; exists {
		return domain.ErrWishlistAlreadyExists
	}
	r.byID[wishlist.ID] = wishlist
	r.byCustID[wishlist.CustomerID] = wishlist.ID
	return nil
}

// Update performs an update on the wishlist. No optimistic locking since
// wishlist doesn't use version field.
func (r *Repository) Update(ctx context.Context, wishlist domain.Wishlist) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.byID[wishlist.ID]
	if !ok {
		return domain.ErrWishlistNotFound
	}
	// Verify customerID hasn't changed
	if existing.CustomerID != wishlist.CustomerID {
		return domain.ErrWishlistNotFound
	}
	r.byID[wishlist.ID] = wishlist
	return nil
}
