package domain

import (
	"context"
	"time"
)

// Repository is the persistence port for orders.
type Repository interface {
	Save(ctx context.Context, o Order) error
	GetByID(ctx context.Context, id string) (Order, error)

	// GetByIdempotencyKey returns the order previously created under key, or
	// ErrOrderNotFound when no order carries it. It is the authoritative
	// idempotency lookup, independent of any cache.
	GetByIdempotencyKey(ctx context.Context, key string) (Order, error)

	// Update persists a change to an existing order's state as a compare-and-set
	// against expected - the status the caller read before mutating it. Writing
	// with a stale expected status writes nothing and reports
	// ErrOrderStateChanged, so a concurrent transition is never clobbered;
	// a missing order reports ErrOrderNotFound.
	Update(ctx context.Context, o Order, expected Status) error

	// ListPlacedBefore returns placed orders created at or before before, oldest
	// first, up to limit. It powers the automatic cancellation of unpaid orders.
	ListPlacedBefore(ctx context.Context, before time.Time, limit int) ([]Order, error)

	// ListByUserID returns one user's orders, newest first. limit must be >= 1
	// and offset >= 0.
	ListByUserID(ctx context.Context, userID string, limit, offset int) ([]Order, error)

	// CountByUserID returns the total number of orders owned by userID.
	CountByUserID(ctx context.Context, userID string) (int, error)
}
