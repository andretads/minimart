package memory

import (
	"context"
	"minimart/internal/customer/domain"
	"sync"
	"time"
)

type Repository struct {
	mu        sync.RWMutex
	customers map[string]domain.Customer
	byUserID  map[string]domain.Customer
}

var _ domain.Repository = (*Repository)(nil)

func New() *Repository {
	return &Repository{
		customers: map[string]domain.Customer{},
		byUserID:  map[string]domain.Customer{},
	}
}

func (r *Repository) GetByID(_ context.Context, id string) (domain.Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.customers[id]
	if !ok {
		return domain.Customer{}, domain.ErrCustomerNotFound
	}
	return c, nil
}

func (r *Repository) GetByUserID(_ context.Context, userID string) (domain.Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.byUserID[userID]
	if !ok {
		return domain.Customer{}, domain.ErrCustomerNotFound
	}
	return c, nil
}

func (r *Repository) Save(_ context.Context, c domain.Customer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byUserID[c.UserID]; ok {
		return domain.ErrCustomerExists
	}
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	if c.Version == 0 {
		c.Version = 1
	}
	r.customers[c.ID] = c
	r.byUserID[c.UserID] = c
	return nil
}

// UpdateProfile updates the customer's display_name under optimistic lock.
// The expected version is carried on the entity's Version field.
// Returns the updated customer with the new version and updated_at.
// Stale version reports ErrCustomerOptimisticLock; missing id reports
// ErrCustomerNotFound.
func (r *Repository) UpdateProfile(_ context.Context, c domain.Customer) (domain.Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.customers[c.ID]
	if !ok {
		return domain.Customer{}, domain.ErrCustomerNotFound
	}
	if stored.Version != c.Version {
		return domain.Customer{}, domain.ErrCustomerOptimisticLock
	}
	c.Version = stored.Version + 1
	c.CreatedAt = stored.CreatedAt
	c.UpdatedAt = time.Now().UTC()
	r.customers[c.ID] = c
	r.byUserID[c.UserID] = c
	return c, nil
}

// Deactivate marks the customer as deactivated and bumps the version.
// It is idempotent: re-deactivating an already-deactivated customer is a no-op
// that preserves the original deactivated_at timestamp but still bumps the version.
// Returns the updated customer with the new version, updated_at, and deactivated_at.
// Missing id reports ErrCustomerNotFound.
func (r *Repository) Deactivate(_ context.Context, c domain.Customer) (domain.Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.customers[c.ID]
	if !ok {
		return domain.Customer{}, domain.ErrCustomerNotFound
	}
	// Domain method is idempotent; we always bump version to mirror the SQL behavior.
	now := time.Now().UTC()
	stored.Deactivate()
	stored.Version++
	stored.UpdatedAt = now
	r.customers[c.ID] = stored
	r.byUserID[stored.UserID] = stored
	return stored, nil
}
