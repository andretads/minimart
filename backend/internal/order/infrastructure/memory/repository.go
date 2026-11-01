package memory

import (
	"context"
	"minimart/internal/order/domain"
	"sort"
	"sync"
	"time"
)

type Repository struct {
	mu     sync.RWMutex
	orders map[string]domain.Order
	byKey  map[string]string
}

var _ domain.Repository = (*Repository)(nil)

func New() *Repository {
	return &Repository{
		orders: make(map[string]domain.Order),
		byKey:  make(map[string]string),
	}
}

// Save stores an order, enforcing the idempotency-key uniqueness the database
// enforces with its partial unique index. An empty key never conflicts.
func (r *Repository) Save(ctx context.Context, o domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o.IdempotencyKey != "" {
		if _, ok := r.byKey[o.IdempotencyKey]; ok {
			return domain.ErrIdempotencyConflict
		}
		r.byKey[o.IdempotencyKey] = o.ID
	}
	r.orders[o.ID] = o
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.orders[id]
	if !ok {
		return domain.Order{}, domain.ErrOrderNotFound
	}
	return o, nil
}

// GetByIdempotencyKey returns the order stored under key, or ErrOrderNotFound.
func (r *Repository) GetByIdempotencyKey(ctx context.Context, key string) (domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, o := range r.orders {
		if o.IdempotencyKey != "" && o.IdempotencyKey == key {
			return o, nil
		}
	}
	return domain.Order{}, domain.ErrOrderNotFound
}

// Update applies a compare-and-set on the stored order's status, mirroring the
// Postgres adapter: a missing order reports ErrOrderNotFound, and a status that no
// longer matches expected reports ErrOrderStateChanged.
func (r *Repository) Update(ctx context.Context, o domain.Order, expected domain.Status) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.orders[o.ID]
	if !ok {
		return domain.ErrOrderNotFound
	}
	if current.Status != expected {
		return domain.ErrOrderStateChanged
	}
	r.orders[o.ID] = o
	return nil
}

// ListByUserID returns one user's orders newest first, mirroring the Postgres
// adapter's ordering and pagination.
func (r *Repository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orders := make([]domain.Order, 0)
	for _, o := range r.orders {
		if o.UserID == userID {
			orders = append(orders, o)
		}
	}
	sort.Slice(orders, func(i, j int) bool {
		if orders[i].CreatedAt.Equal(orders[j].CreatedAt) {
			return orders[i].ID > orders[j].ID
		}
		return orders[i].CreatedAt.After(orders[j].CreatedAt)
	})

	if offset >= len(orders) {
		return []domain.Order{}, nil
	}
	end := min(offset+limit, len(orders))
	return orders[offset:end], nil
}

// CountByUserID returns the number of orders owned by userID.
func (r *Repository) CountByUserID(ctx context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, o := range r.orders {
		if o.UserID == userID {
			count++
		}
	}
	return count, nil
}

// ListPlacedBefore returns placed orders created at or before before, oldest
// first, mirroring the Postgres adapter.
func (r *Repository) ListPlacedBefore(ctx context.Context, before time.Time, limit int) ([]domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orders := make([]domain.Order, 0)
	for _, o := range r.orders {
		if o.Status == domain.StatusPlaced && !o.CreatedAt.After(before) {
			orders = append(orders, o)
		}
	}
	sort.Slice(orders, func(i, j int) bool {
		if orders[i].CreatedAt.Equal(orders[j].CreatedAt) {
			return orders[i].ID < orders[j].ID
		}
		return orders[i].CreatedAt.Before(orders[j].CreatedAt)
	})
	if limit > 0 && len(orders) > limit {
		orders = orders[:limit]
	}
	if orders == nil {
		orders = []domain.Order{}
	}
	return orders, nil
}
