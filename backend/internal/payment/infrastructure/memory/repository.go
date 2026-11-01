// Package memory provides an in-memory domain.Repository for tests that do not
// need a database (for example the HTTP handler tests).
package memory

import (
	"context"
	"minimart/internal/payment/domain"
	"sync"
	"time"
)

// Repository is a thread-safe in-memory payment store.
type Repository struct {
	mu       sync.RWMutex
	payments map[string]domain.Payment
	byKey    map[string]string
}

var _ domain.Repository = (*Repository)(nil)

// New returns an empty repository.
func New() *Repository {
	return &Repository{
		payments: make(map[string]domain.Payment),
		byKey:    make(map[string]string),
	}
}

// Save stores a payment, enforcing the idempotency-key uniqueness the database
// enforces with its unique index.
func (r *Repository) Save(_ context.Context, p domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byKey[p.IdempotencyKey]; ok {
		return domain.ErrIdempotencyConflict
	}
	r.payments[p.ID] = p
	r.byKey[p.IdempotencyKey] = p.ID
	return nil
}

// GetByID returns the payment with the given id.
func (r *Repository) GetByID(_ context.Context, id string) (domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.payments[id]
	if !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}
	return p, nil
}

// GetByIdempotencyKey returns the payment stored under the given key.
func (r *Repository) GetByIdempotencyKey(_ context.Context, key string) (domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byKey[key]
	if !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}
	return r.payments[id], nil
}

// Update finalizes a pending payment, mirroring the database's compare-and-set.
func (r *Repository) Update(_ context.Context, p domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.payments[p.ID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	if existing.Status != domain.StatusPending {
		return domain.ErrPaymentNotPending
	}
	r.payments[p.ID] = p
	return nil
}

// ListReconcilable returns pending or unknown payments older than updatedBefore.
func (r *Repository) ListReconcilable(_ context.Context, updatedBefore time.Time, limit int) ([]domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []domain.Payment
	for _, p := range r.payments {
		if (p.Status == domain.StatusPending || p.Status == domain.StatusUnknown) && p.UpdatedAt.Before(updatedBefore) {
			out = append(out, p)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

// Reconcile persists a reconciled terminal state for a pending/unknown payment.
func (r *Repository) Reconcile(_ context.Context, p domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.payments[p.ID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	if existing.Status != domain.StatusPending && existing.Status != domain.StatusUnknown {
		return domain.ErrPaymentNotPending
	}
	r.payments[p.ID] = p
	return nil
}

// Refund persists a refunded payment, mirroring the database's compare-and-set.
func (r *Repository) Refund(_ context.Context, p domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.payments[p.ID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	switch existing.Status {
	case domain.StatusPending, domain.StatusSucceeded, domain.StatusUnknown:
	default:
		return domain.ErrPaymentNotRefundable
	}
	r.payments[p.ID] = p
	return nil
}

// ListSucceededByOrderID returns the captured payments for an order.
func (r *Repository) ListSucceededByOrderID(_ context.Context, orderID string) ([]domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []domain.Payment
	for _, p := range r.payments {
		if p.OrderID == orderID && p.Status == domain.StatusSucceeded {
			out = append(out, p)
		}
	}
	return out, nil
}
