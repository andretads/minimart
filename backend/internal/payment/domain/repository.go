package domain

import (
	"context"
	"time"
)

// Repository is the persistence port for payments. Save and Update write the
// payment's domain events to the transactional outbox in the same transaction,
// so the state change and its events commit atomically.
type Repository interface {
	Save(ctx context.Context, p Payment) error
	GetByID(ctx context.Context, id string) (Payment, error)
	// GetByIdempotencyKey returns the payment already created for key, or
	// ErrPaymentNotFound. It is the authoritative fallback when the Redis
	// idempotency cache misses or is unavailable.
	GetByIdempotencyKey(ctx context.Context, key string) (Payment, error)
	// Update persists a payment's terminal state, but only while it is still
	// pending (compare-and-set). It reports ErrPaymentNotPending when the row
	// has already been finalized.
	Update(ctx context.Context, p Payment) error
	// ListReconcilable returns pending or unknown payments older than updatedBefore.
	ListReconcilable(ctx context.Context, updatedBefore time.Time, limit int) ([]Payment, error)
	// Reconcile persists a reconciled terminal state for a payment that is still
	// pending or unknown. It reports ErrPaymentNotPending when the row has
	// already been finalized.
	Reconcile(ctx context.Context, p Payment) error
	// Refund persists a refunded payment, but only while it is still pending or
	// succeeded (compare-and-set), writing its events in the same transaction.
	// It reports ErrPaymentNotRefundable when the row can no longer be refunded.
	Refund(ctx context.Context, p Payment) error
	// ListSucceededByOrderID returns the captured payments for an order, which
	// are the ones a cancellation must refund.
	ListSucceededByOrderID(ctx context.Context, orderID string) ([]Payment, error)
}
