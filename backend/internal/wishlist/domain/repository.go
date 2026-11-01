package domain

import "context"

// Repository is the persistence port the wishlist domain needs.
type Repository interface {
	// GetByCustomerID returns the wishlist for the given customer.
	// Returns ErrWishlistNotFound when no row exists.
	GetByCustomerID(ctx context.Context, customerID string) (Wishlist, error)

	// Create persists a new wishlist.
	// Returns ErrWishlistAlreadyExists if a wishlist for the customer already exists.
	Create(ctx context.Context, w Wishlist) error

	// Update persists changes to an existing wishlist.
	Update(ctx context.Context, w Wishlist) error
}
