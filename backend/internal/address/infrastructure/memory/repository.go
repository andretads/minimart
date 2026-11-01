package memory

import (
	"context"
	"minimart/internal/address/domain"
	"slices"
	"sync"
	"time"
)

type Repository struct {
	mu        sync.RWMutex
	addresses map[string]domain.Address
}

var _ domain.Repository = (*Repository)(nil)

func New() *Repository {
	return &Repository{
		addresses: map[string]domain.Address{},
	}
}

func (r *Repository) GetByID(_ context.Context, id string) (domain.Address, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.addresses[id]
	if !ok {
		return domain.Address{}, domain.ErrAddressNotFound
	}
	return a, nil
}

func (r *Repository) GetByCustomerID(_ context.Context, customerID string) ([]domain.Address, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Address, 0)
	for _, a := range r.addresses {
		if a.CustomerID == customerID {
			out = append(out, a)
		}
	}
	slices.SortFunc(out, func(a, b domain.Address) int {
		if a.IsDefault != b.IsDefault {
			if a.IsDefault {
				return -1
			}
			return 1
		}
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return out, nil
}

func (r *Repository) GetDefaultByCustomerID(_ context.Context, customerID string) (domain.Address, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.addresses {
		if a.CustomerID == customerID && a.IsDefault {
			return a, nil
		}
	}
	return domain.Address{}, domain.ErrAddressNotFound
}

func (r *Repository) Save(_ context.Context, a domain.Address) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.addresses[a.ID]; ok {
		return domain.ErrAddressExists
	}
	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}
	if a.Version == 0 {
		a.Version = 1
	}
	r.addresses[a.ID] = a
	return nil
}

func (r *Repository) Update(_ context.Context, a domain.Address) (domain.Address, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.addresses[a.ID]
	if !ok {
		return domain.Address{}, domain.ErrAddressNotFound
	}
	if stored.Version != a.Version {
		return domain.Address{}, domain.ErrInvalidVersion
	}
	a.Version = stored.Version + 1
	a.CreatedAt = stored.CreatedAt
	a.UpdatedAt = time.Now().UTC()
	r.addresses[a.ID] = a
	return a, nil
}

func (r *Repository) Delete(_ context.Context, id, customerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.addresses[id]
	if !ok {
		return domain.ErrAddressNotFound
	}
	if stored.CustomerID != customerID {
		return domain.ErrAddressNotFound
	}
	delete(r.addresses, id)
	return nil
}

func (r *Repository) SetDefault(_ context.Context, id, customerID string) (domain.Address, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if the address exists and belongs to the customer
	stored, ok := r.addresses[id]
	if !ok {
		return domain.Address{}, domain.ErrAddressNotFound
	}
	if stored.CustomerID != customerID {
		return domain.Address{}, domain.ErrAddressNotFound
	}

	// Unset old default for this customer
	for k, a := range r.addresses {
		if a.CustomerID == customerID && a.IsDefault {
			a.IsDefault = false
			a.Version++
			a.UpdatedAt = time.Now().UTC()
			r.addresses[k] = a
		}
	}

	// Set new default
	stored.IsDefault = true
	stored.Version++
	stored.UpdatedAt = time.Now().UTC()
	r.addresses[id] = stored

	return stored, nil
}
