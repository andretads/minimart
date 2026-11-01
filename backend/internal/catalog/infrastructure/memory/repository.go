// Package memory is an in-memory adapter for the catalog Repository port. Data
// lives only for the life of the process. It is the simplest possible adapter and
// stays useful forever as a fast, dependency-free double in tests.
package memory

import (
	"context"
	"minimart/internal/catalog/domain"
	"minimart/pkg/moderation"
	"sort"
	"sync"
	"time"
)

// Repository stores products in a map guarded by a mutex, safe for concurrent use.
// reservations tracks outstanding stock holds per order (order id -> product id ->
// quantity); a product key disappears when its hold is released, which is what
// makes release idempotent.
type Repository struct {
	mu              sync.RWMutex
	products        map[string]domain.Product
	reservations    map[string]map[string]int
	reviews         map[string][]domain.Review       // productID -> reviews
	images          map[string][]domain.ProductImage // productID -> images
	ratingSummaries map[string]domain.RatingSummary  // productID -> summary
}

// Compile-time proof that Repository satisfies the port. If a method drifts, the
// build fails here, not at runtime.
var _ domain.Repository = (*Repository)(nil)

func New() *Repository {
	return &Repository{
		products:        make(map[string]domain.Product),
		reservations:    make(map[string]map[string]int),
		reviews:         make(map[string][]domain.Review),
		images:          make(map[string][]domain.ProductImage),
		ratingSummaries: make(map[string]domain.RatingSummary),
	}
}

// Save stores the product, upserting by id. On a collision it keeps the stored
// version and created_at and refreshes updated_at - the same thing the Postgres
// ON CONFLICT branch does - so both adapters behave alike.
func (r *Repository) Save(ctx context.Context, p domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if stored, ok := r.products[p.ID]; ok {
		p.Version = stored.Version
		p.CreatedAt = stored.CreatedAt
		p.UpdatedAt = time.Now().UTC()
	}
	r.products[p.ID] = p
	return nil
}

// Update applies p only when its version still matches the stored one, bumping
// the version exactly like the SQL "version = version + 1" does and keeping the
// original created_at. It reports ErrProductOptimisticLock for a stale version
// and ErrProductNotFound for an unknown id, and returns the stored product.
func (r *Repository) Update(ctx context.Context, p domain.Product) (domain.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.products[p.ID]
	if !ok {
		return domain.Product{}, domain.ErrProductNotFound
	}
	if stored.Version != p.Version {
		return domain.Product{}, domain.ErrProductOptimisticLock
	}
	p.Version = stored.Version + 1
	p.CreatedAt = stored.CreatedAt
	p.UpdatedAt = time.Now().UTC()
	r.products[p.ID] = p
	return p, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.products[id]
	if !ok {
		return domain.Product{}, domain.ErrProductNotFound
	}
	return p, nil
}

func (r *Repository) List(ctx context.Context) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Product, 0, len(r.products))
	for _, p := range r.products {
		out = append(out, p)
	}
	return out, nil
}

// ListByCategory returns all products in the given category.
func (r *Repository) ListByCategory(ctx context.Context, categoryID string) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Product, 0, len(r.products))
	for _, p := range r.products {
		if p.CategoryID == categoryID {
			out = append(out, p)
		}
	}
	return out, nil
}

// ReserveStock validates every item before mutating anything, so a shortage on
// any item leaves stock untouched - the same all-or-nothing guarantee the SQL
// transaction gives. Items already reserved for this order are skipped, making a
// retried reservation a no-op.
func (r *Repository) ReserveStock(_ context.Context, orderID string, items []domain.ReservationItem) error {
	if len(items) == 0 {
		return domain.ErrInvalidReservation
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	outstanding := r.reservations[orderID]
	for _, it := range items {
		if it.Quantity <= 0 {
			return domain.ErrInvalidReservation
		}
		if outstanding != nil && outstanding[it.ProductID] > 0 {
			continue // already reserved for this order
		}
		p, ok := r.products[it.ProductID]
		if !ok {
			return domain.ErrProductNotFound
		}
		if it.Quantity > p.Stock {
			return domain.ErrInsufficientStock
		}
	}

	if outstanding == nil {
		outstanding = make(map[string]int)
		r.reservations[orderID] = outstanding
	}
	for _, it := range items {
		if outstanding[it.ProductID] > 0 {
			continue
		}
		p := r.products[it.ProductID]
		p.Stock -= it.Quantity
		p.Version++
		p.UpdatedAt = time.Now().UTC()
		r.products[it.ProductID] = p
		outstanding[it.ProductID] = it.Quantity
	}
	return nil
}

// ReleaseStock restores every outstanding reservation for the order and forgets
// them. A second call finds nothing to release, so it is a no-op.
func (r *Repository) ReleaseStock(_ context.Context, orderID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	outstanding := r.reservations[orderID]
	for productID, qty := range outstanding {
		p, ok := r.products[productID]
		if !ok {
			continue
		}
		p.Stock += qty
		p.Version++
		p.UpdatedAt = time.Now().UTC()
		r.products[productID] = p
	}
	delete(r.reservations, orderID)
	return nil
}

// SaveReview persists a new review for a product.
func (r *Repository) SaveReview(ctx context.Context, rev domain.Review) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.reviews[rev.ProductID] {
		if existing.CustomerID == rev.CustomerID {
			return domain.ErrDuplicateReview
		}
	}
	r.reviews[rev.ProductID] = append(r.reviews[rev.ProductID], rev)
	return nil
}

// GetReviews returns all approved reviews for a product.
func (r *Repository) GetReviews(ctx context.Context, productID string) ([]domain.Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	allReviews := r.reviews[productID]
	out := make([]domain.Review, 0, len(allReviews))
	for _, rev := range allReviews {
		if rev.Status == "approved" {
			out = append(out, rev)
		}
	}
	return out, nil
}

// GetReviewsPaginated returns a page of approved reviews, newest first.
func (r *Repository) GetReviewsPaginated(ctx context.Context, productID string, limit, offset int) ([]domain.Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	allReviews := make([]domain.Review, 0)
	for _, rev := range r.reviews[productID] {
		if rev.Status == "approved" {
			allReviews = append(allReviews, rev)
		}
	}
	// sort newest first
	sort.SliceStable(allReviews, func(i, j int) bool { return allReviews[i].CreatedAt.After(allReviews[j].CreatedAt) })
	if offset >= len(allReviews) {
		return []domain.Review{}, nil
	}
	end := min(offset+limit, len(allReviews))
	out := make([]domain.Review, end-offset)
	copy(out, allReviews[offset:end])
	return out, nil
}

// CountReviews returns the number of approved reviews.
func (r *Repository) CountReviews(ctx context.Context, productID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, rev := range r.reviews[productID] {
		if rev.Status == "approved" {
			count++
		}
	}
	return count, nil
}

// GetRatingSummary returns the rating summary.
func (r *Repository) GetRatingSummary(ctx context.Context, productID string) (domain.RatingSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.ratingSummaries[productID]; ok {
		return s, nil
	}
	// compute from reviews
	var sum int64
	var count int64
	for _, rev := range r.reviews[productID] {
		if rev.Status == "approved" {
			sum += int64(rev.Rating)
			count++
		}
	}
	avg := 0.0
	if count > 0 {
		avg = float64(sum) / float64(count)
	}
	return domain.RatingSummary{ProductID: productID, TotalReviews: count, RatingSum: sum, AverageRating: avg}, nil
}

// UpdateRatingSummary updates the rating summary.
func (r *Repository) UpdateRatingSummary(ctx context.Context, productID string, ratingSum, totalReviews int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	avg := 0.0
	if totalReviews > 0 {
		avg = float64(ratingSum) / float64(totalReviews)
	}
	r.ratingSummaries[productID] = domain.RatingSummary{
		ProductID:     productID,
		TotalReviews:  totalReviews,
		RatingSum:     ratingSum,
		AverageRating: avg,
		UpdatedAt:     time.Now().UTC(),
	}
	return nil
}

// GetReviewByID returns a review by its ID.
func (r *Repository) GetReviewByID(ctx context.Context, id string) (domain.Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, reviews := range r.reviews {
		for _, rev := range reviews {
			if rev.ID == id {
				return rev, nil
			}
		}
	}
	return domain.Review{}, domain.ErrProductReviewNotFound
}

// UpdateReviewStats updates the denormalized review counters on the product.
func (r *Repository) UpdateReviewStats(ctx context.Context, productID string, totalRating, totalReviews int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.products[productID]
	if !ok {
		return domain.ErrProductNotFound
	}
	p.TotalRating = totalRating
	p.TotalReviews = totalReviews
	p.Version++
	p.UpdatedAt = time.Now().UTC()
	r.products[productID] = p
	return nil
}

// UpdateReviewModeration updates the moderation fields of a review.
func (r *Repository) UpdateReviewModeration(ctx context.Context, reviewID string, result moderation.Result) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for productID, reviews := range r.reviews {
		for i, rev := range reviews {
			if rev.ID == reviewID {
				now := time.Now().UTC()
				rev.Status = result.Decision
				rev.ModeratedAt = &now
				rev.ModerationDecision = result.Decision
				rev.ModerationConfidence = result.Confidence
				rev.ModerationCategories = result.Categories
				rev.ModerationExplanation = result.Explanation
				rev.ModerationModel = result.Model
				rev.ModerationModelVersion = result.ModelVersion
				r.reviews[productID][i] = rev
				return nil
			}
		}
	}
	return domain.ErrProductReviewNotFound
}

// SaveProductImage appends an image, assigns the next position, and bumps the
// owning product's version.
func (r *Repository) SaveProductImage(_ context.Context, img domain.ProductImage) (domain.ProductImage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.products[img.ProductID]
	if !ok {
		return domain.ProductImage{}, domain.ErrProductNotFound
	}

	position := 0
	for _, existing := range r.images[img.ProductID] {
		if existing.Position >= position {
			position = existing.Position + 1
		}
	}
	img.Position = position
	img.Version = 1
	now := time.Now().UTC()
	if img.CreatedAt.IsZero() {
		img.CreatedAt = now
	}
	img.UpdatedAt = now
	r.images[img.ProductID] = append(r.images[img.ProductID], img)

	p.Version++
	p.UpdatedAt = now
	r.products[p.ID] = p
	return img, nil
}

// DeleteProductImage removes an image and bumps the owning product's version.
func (r *Repository) DeleteProductImage(_ context.Context, productID, imageID string) (domain.ProductImage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.products[productID]
	if !ok {
		return domain.ProductImage{}, domain.ErrProductNotFound
	}

	images := r.images[productID]
	for i, img := range images {
		if img.ID != imageID {
			continue
		}
		r.images[productID] = append(images[:i:i], images[i+1:]...)
		p.Version++
		p.UpdatedAt = time.Now().UTC()
		r.products[p.ID] = p
		return img, nil
	}
	return domain.ProductImage{}, domain.ErrProductImageNotFound
}

// ListProductImages returns a product's images ordered by position.
func (r *Repository) ListProductImages(_ context.Context, productID string) ([]domain.ProductImage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := append([]domain.ProductImage(nil), r.images[productID]...)
	sortImages(out)
	return out, nil
}

// ListProductImagesByProductIDs returns images grouped by product ID.
func (r *Repository) ListProductImagesByProductIDs(_ context.Context, productIDs []string) (map[string][]domain.ProductImage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string][]domain.ProductImage, len(productIDs))
	for _, productID := range productIDs {
		images := append([]domain.ProductImage(nil), r.images[productID]...)
		sortImages(images)
		out[productID] = images
	}
	return out, nil
}

func sortImages(images []domain.ProductImage) {
	sort.SliceStable(images, func(i, j int) bool {
		if images[i].Position != images[j].Position {
			return images[i].Position < images[j].Position
		}
		return images[i].CreatedAt.Before(images[j].CreatedAt)
	})
}
