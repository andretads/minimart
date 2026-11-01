package domain

import (
	"context"
	"minimart/pkg/moderation"
)

// Repository is the persistence port the catalog domain needs. The domain owns
// this interface; adapters under infrastructure/ implement it (in-memory now,
// Postgres later). ctx is the first parameter of every method from the start -
// the shape a real database adapter requires.
type Repository interface {
	Save(ctx context.Context, p Product) error
	// Update persists a changed product under its optimistic lock: it only
	// applies when the stored version still equals p.Version, and returns the
	// stored product with its new version and updated_at. A stale version
	// repo`rts ErrProductOptimisticLock; a missing id reports ErrProductNotFound.
	Update(ctx context.Context, p Product) (Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
	List(ctx context.Context) ([]Product, error)
	// ListByCategory returns all products in the given category.
	ListByCategory(ctx context.Context, categoryID string) ([]Product, error)

	// ReserveStock atomically reserves each item's quantity for orderID. It is
	// all-or-nothing: if any item lacks stock the whole reservation is rolled
	// back and ErrInsufficientStock (or ErrProductNotFound) is reported. Calling
	// it again for the same order is a no-op, so a retried placement never
	// double-reserves.
	ReserveStock(ctx context.Context, orderID string, items []ReservationItem) error
	// ReleaseStock restores every not-yet-released reservation for orderID. It is
	// idempotent: releasing an order with nothing outstanding writes nothing, so
	// a redelivered release event is safe.
	ReleaseStock(ctx context.Context, orderID string) error

	// SaveReview persists a new review for a product.
	SaveReview(ctx context.Context, r Review) error
	// GetReviews returns all approved reviews for a product.
	GetReviews(ctx context.Context, productID string) ([]Review, error)
	// GetReviewsPaginated returns a page of approved reviews for a product, newest first.
	GetReviewsPaginated(ctx context.Context, productID string, limit, offset int) ([]Review, error)
	// CountReviews returns the number of approved reviews for a product.
	CountReviews(ctx context.Context, productID string) (int, error)
	// GetReviewByID returns a review by its ID.
	GetReviewByID(ctx context.Context, id string) (Review, error)
	// UpdateReviewModeration updates the moderation fields of a review.
	UpdateReviewModeration(ctx context.Context, reviewID string, result moderation.Result) error
	// UpdateReviewStats updates the denormalized review counters on the product.
	UpdateReviewStats(ctx context.Context, productID string, totalRating, totalReviews int64) error
	// GetRatingSummary returns the rating summary for a product.
	GetRatingSummary(ctx context.Context, productID string) (RatingSummary, error)
	// UpdateRatingSummary updates the rating summary aggregate for a product.
	UpdateRatingSummary(ctx context.Context, productID string, ratingSum, totalReviews int64) error

	// SaveProductImage inserts an image for a product, assigns its position,
	// and bumps the owning product's version atomically. It reports
	// ErrProductNotFound when the product does not exist.
	SaveProductImage(ctx context.Context, img ProductImage) (ProductImage, error)
	// DeleteProductImage removes an image and bumps the owning product's version
	// atomically. It reports ErrProductImageNotFound when the image does not
	// belong to the product, and ErrProductNotFound when the product is missing.
	DeleteProductImage(ctx context.Context, productID, imageID string) (ProductImage, error)
	// ListProductImages returns a product's images ordered by position.
	ListProductImages(ctx context.Context, productID string) ([]ProductImage, error)
	// ListProductImagesByProductIDs returns images grouped by product ID, each
	// group ordered by position.
	ListProductImagesByProductIDs(ctx context.Context, productIDs []string) (map[string][]ProductImage, error)
}
