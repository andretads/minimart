package domain

import "context"

// Repository is the persistence port for customers. The domain owns this
// interface; adapters under infrastructure/ implement it.
type Repository interface {
	// GetByID returns the customer by ID, or ErrCustomerNotFound.
	GetByID(ctx context.Context, id string) (Customer, error)

	// GetByUserID returns the customer by user ID, or ErrCustomerNotFound.
	GetByUserID(ctx context.Context, userID string) (Customer, error)

	// Save inserts a new customer. It reports ErrCustomerExists if a customer
	// with the same user_id already exists (enforced by a unique index).
	Save(ctx context.Context, c Customer) error

	// UpdateProfile updates the display_name for an existing customer under
	// optimistic lock: the caller provides the expected version on the entity.
	// It returns the persisted customer with the new version and updated_at.
	// Stale version reports ErrCustomerOptimisticLock; missing id reports
	// ErrCustomerNotFound.
	UpdateProfile(ctx context.Context, c Customer) (Customer, error)

	// Deactivate marks a customer as deactivated. It bumps the version and
	// returns the persisted customer with the new version, updated_at, and
	// deactivated_at. It is idempotent: re-deactivating an already-deactivated
	// customer is a no-op that preserves the original deactivated_at timestamp
	// but still bumps the version. It reports ErrCustomerNotFound if the
	// customer does not exist.
	Deactivate(ctx context.Context, c Customer) (Customer, error)
}
