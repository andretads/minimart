package domain

import "context"

// Repository is the persistence port for categories. The domain owns this
// interface; adapters under infrastructure/ implement it.
type Repository interface {
	// GetByID returns the category by ID, or ErrCategoryNotFound.
	GetByID(ctx context.Context, id string) (Category, error)

	// List returns all categories ordered by lower(name), id.
	List(ctx context.Context) ([]Category, error)

	// Save inserts a new category. It reports ErrCategoryExists if a category
	// with the same name (case-insensitive) already exists.
	Save(ctx context.Context, c Category) error

	// Update updates the name and icon for an existing category under
	// optimistic lock: the caller provides the expected version on the entity.
	// It returns the persisted category with the new version and updated_at.
	// Stale version reports ErrCategoryOptimisticLock; missing id reports
	// ErrCategoryNotFound.
	Update(ctx context.Context, c Category) (Category, error)

	// Delete deletes a category by ID under optimistic lock.
	// It reports ErrCategoryOptimisticLock if the version does not match,
	// and ErrCategoryNotFound if the category does not exist.
	Delete(ctx context.Context, id string, version int) error
}
