package domain

import "context"

// Repository is the persistence port for addresses. The domain owns this
// interface; adapters under infrastructure/ implement it.
type Repository interface {
	// GetByID returns the address by ID, or ErrAddressNotFound.
	GetByID(ctx context.Context, id string) (Address, error)

	// GetByCustomerID returns all addresses for a customer, ordered by
	// is_default DESC, created_at DESC. Returns empty slice if none.
	GetByCustomerID(ctx context.Context, customerID string) ([]Address, error)

	// GetDefaultByCustomerID returns the default address for a customer,
	// or ErrAddressNotFound if none exists.
	GetDefaultByCustomerID(ctx context.Context, customerID string) (Address, error)

	// Save inserts a new address. It reports ErrAddressExists if an address
	// with the same ID already exists.
	Save(ctx context.Context, a Address) error

	// Update updates an existing address under optimistic lock: the caller
	// provides the expected version on the entity. It returns the persisted
	// address with the new version and updated_at. Stale version reports
	// ErrInvalidVersion; missing id reports ErrAddressNotFound.
	Update(ctx context.Context, a Address) (Address, error)

	// Delete removes an address by ID. It reports ErrAddressNotFound if the
	// address does not exist or does not belong to the customer.
	Delete(ctx context.Context, id, customerID string) error

	// SetDefault marks an address as the default for its customer, unsetting
	// any existing default in the same transaction. It returns the persisted
	// address with the new version, updated_at, and is_default=true. It reports
	// ErrAddressNotFound if the address does not exist or does not belong to
	// the customer.
	SetDefault(ctx context.Context, id, customerID string) (Address, error)
}
