package memory

import (
	"context"
	"minimart/internal/showcase/domain"
	"sort"
	"strings"
	"sync"
)

// Repository is an in-memory implementation of domain.Repository for testing.
type Repository struct {
	mu         sync.RWMutex
	products   map[string]domain.Product
	images     map[string][]domain.ProductImage
	reviews    map[string][]domain.Review
	categories map[string]domain.Category
}

func New() *Repository {
	return &Repository{
		products:   make(map[string]domain.Product),
		images:     make(map[string][]domain.ProductImage),
		reviews:    make(map[string][]domain.Review),
		categories: make(map[string]domain.Category),
	}
}

func (r *Repository) Save(p domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[p.ID] = p
	return nil
}

// SaveImage appends an image to a product's image list.
func (r *Repository) SaveImage(img domain.ProductImage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.images[img.ProductID] = append(r.images[img.ProductID], img)
	return nil
}

// SaveReview appends a review to a product's review list.
func (r *Repository) SaveReview(review domain.Review) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reviews[review.ProductID] = append(r.reviews[review.ProductID], review)
	return nil
}

func (r *Repository) List(ctx context.Context, limit int, after *domain.Cursor) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	products := make([]domain.Product, 0, len(r.products))
	for _, p := range r.products {
		products = append(products, p)
	}
	sort.Slice(products, func(i, j int) bool {
		if !products[i].CreatedAt.Equal(products[j].CreatedAt) {
			return products[i].CreatedAt.After(products[j].CreatedAt)
		}
		return products[i].ID > products[j].ID
	})

	if after != nil {
		filtered := make([]domain.Product, 0, len(products))
		for _, p := range products {
			if p.CreatedAt.Before(after.CreatedAt) ||
				(p.CreatedAt.Equal(after.CreatedAt) && p.ID < after.ID) {
				filtered = append(filtered, p)
			}
		}
		products = filtered
	}

	if len(products) > limit {
		products = products[:limit]
	}
	if products == nil {
		products = []domain.Product{}
	}
	return products, nil
}

func (r *Repository) ListByCategory(ctx context.Context, categoryID string, limit int, after *domain.Cursor) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	products := make([]domain.Product, 0, len(r.products))
	for _, p := range r.products {
		if p.CategoryID == categoryID {
			products = append(products, p)
		}
	}
	sort.Slice(products, func(i, j int) bool {
		if !products[i].CreatedAt.Equal(products[j].CreatedAt) {
			return products[i].CreatedAt.After(products[j].CreatedAt)
		}
		return products[i].ID > products[j].ID
	})

	if after != nil {
		filtered := make([]domain.Product, 0, len(products))
		for _, p := range products {
			if p.CreatedAt.Before(after.CreatedAt) ||
				(p.CreatedAt.Equal(after.CreatedAt) && p.ID < after.ID) {
				filtered = append(filtered, p)
			}
		}
		products = filtered
	}

	if len(products) > limit {
		products = products[:limit]
	}
	if products == nil {
		products = []domain.Product{}
	}
	return products, nil
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

// ListImages returns the images of a product ordered by position.
func (r *Repository) ListImages(ctx context.Context, productID string) ([]domain.ProductImage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return sortedImages(r.images[productID]), nil
}

// ListImagesByProductIDs returns images for the given products grouped by
// product ID, ordered by position within each product.
func (r *Repository) ListImagesByProductIDs(ctx context.Context, productIDs []string) (map[string][]domain.ProductImage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string][]domain.ProductImage, len(productIDs))
	for _, productID := range productIDs {
		out[productID] = sortedImages(r.images[productID])
	}
	return out, nil
}

// ListReviews returns a page of approved reviews of a product, newest first.
func (r *Repository) ListReviews(ctx context.Context, productID string, limit, offset int) ([]domain.Review, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]domain.Review, 0)
	for _, review := range r.reviews[productID] {
		if review.Status == "approved" {
			all = append(all, review)
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	if offset >= len(all) {
		return []domain.Review{}, nil
	}
	end := min(offset+limit, len(all))
	out := make([]domain.Review, end-offset)
	copy(out, all[offset:end])
	return out, nil
}

// CountReviews returns the total number of approved reviews of a product.
func (r *Repository) CountReviews(ctx context.Context, productID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, review := range r.reviews[productID] {
		if review.Status == "approved" {
			count++
		}
	}
	return count, nil
}

// ListCategories returns all categories ordered by lower name, id.
func (r *Repository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.Category, 0, len(r.categories))
	for _, c := range r.categories {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		li, lj := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
		if li != lj {
			return li < lj
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// SaveCategory stores a category in memory (test helper).
func (r *Repository) SaveCategory(c domain.Category) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.categories[c.ID] = c
}

func sortedImages(images []domain.ProductImage) []domain.ProductImage {
	out := make([]domain.ProductImage, len(images))
	copy(out, images)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Position < out[j].Position
	})
	return out
}

var _ domain.Repository = (*Repository)(nil)
