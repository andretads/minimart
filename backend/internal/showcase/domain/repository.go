package domain

import "context"

// Repository is the port for reading products. The showcase domain owns its own
// Postgres reader queries over the shared products, product_images, and
// product_reviews tables and does not import the catalog's application or
// infrastructure layers.
type Repository interface {
	// List returns a page of products ordered newest-first, optionally after
	// the given cursor. limit must be >= 1.
	List(ctx context.Context, limit int, after *Cursor) ([]Product, error)

	// ListByCategory returns a page of products in the given category ordered
	// newest-first, optionally after the given cursor. limit must be >= 1.
	ListByCategory(ctx context.Context, categoryID string, limit int, after *Cursor) ([]Product, error)

	// GetByID returns a single product by ID.
	// Returns ErrProductNotFound when no row exists.
	GetByID(ctx context.Context, id string) (Product, error)

	// ListImages returns the images of a product ordered by position.
	ListImages(ctx context.Context, productID string) ([]ProductImage, error)

	// ListImagesByProductIDs returns images for the given products grouped by
	// product ID, ordered by position within each product.
	ListImagesByProductIDs(ctx context.Context, productIDs []string) (map[string][]ProductImage, error)

	// ListReviews returns a page of approved reviews of a product, newest first.
	ListReviews(ctx context.Context, productID string, limit, offset int) ([]Review, error)

	// CountReviews returns the total number of approved reviews of a product.
	CountReviews(ctx context.Context, productID string) (int, error)

	// ListCategories returns all categories ordered by name.
	ListCategories(ctx context.Context) ([]Category, error)
}
