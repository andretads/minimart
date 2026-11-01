package application

import (
	"context"
	"minimart/internal/catalog/domain"
	"minimart/pkg/metrics"
)

// UpdateProductRequest is the input to the UpdateProduct use case.
type UpdateProductRequest struct {
	ID          string
	Name        string
	PriceCents  int64
	Version     int
	CategoryID  string
	SKU         string
	Description string
	Details     []domain.ProductDetail
}

// UpdateProduct changes a product's name, price, category, SKU, description, and details, preserving every other
// field. It reads the existing product first so stock is carried through, then
// persists through the port under the caller-supplied version. The storage
// predicate only matches while the stored version still equals req.Version, so
// a caller that read a stale value gets domain.ErrProductOptimisticLock instead
// of silently clobbering a newer write. A missing or non-positive version is
// rejected up front. CategoryID is required. The returned product reflects the
// stored version and updated_at.
func (s *Service) UpdateProduct(ctx context.Context, req UpdateProductRequest) (domain.Product, error) {
	if req.Version <= 0 {
		s.m.IncProductUpdated(metrics.StatusError)
		return domain.Product{}, domain.ErrInvalidVersion
	}
	if req.CategoryID == "" {
		s.m.IncProductUpdated(metrics.StatusError)
		return domain.Product{}, domain.ErrProductCategoryRequired
	}
	// Validate category exists
	_, err := s.categoryProvider.GetCategory(ctx, req.CategoryID)
	if err != nil {
		s.m.IncProductUpdated(metrics.StatusError)
		return domain.Product{}, err
	}

	existing, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		s.m.IncProductUpdated(metrics.StatusError)
		return domain.Product{}, err
	}
	updated, err := existing.UpdateNameAndPrice(req.Name, req.PriceCents)
	if err != nil {
		s.m.IncProductUpdated(metrics.StatusError)
		return domain.Product{}, err
	}

	// Update category
	updated, err = updated.UpdateCategory(req.CategoryID)
	if err != nil {
		s.m.IncProductUpdated(metrics.StatusError)
		return domain.Product{}, err
	}

	// Update SKU
	updated, err = updated.UpdateSKU(req.SKU)
	if err != nil {
		s.m.IncProductUpdated(metrics.StatusError)
		return domain.Product{}, err
	}

	// Update description
	updated = updated.UpdateDescription(req.Description)

	// Update details
	updated = updated.UpdateDetails(req.Details)

	updated.Version = req.Version
	persisted, err := s.repo.Update(ctx, updated)
	if err != nil {
		s.m.IncProductUpdated(metrics.StatusError)
		return domain.Product{}, err
	}
	s.m.IncProductUpdated(metrics.StatusSuccess)
	return persisted, nil
}
