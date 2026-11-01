package application

import (
	"context"
	"minimart/internal/category/domain"
	"minimart/pkg/metrics"
)

// UpdateCategoryRequest carries the input for updating a category.
// Version is the optimistic lock token the caller read from a prior GET; it must be positive.
type UpdateCategoryRequest struct {
	ID      string
	Name    string
	Icon    string
	Version int
}

// UpdateCategory updates a category's name and icon. It loads the category,
// applies the domain-level validation via UpdateNameAndIcon, then overwrites
// the loaded entity's Version with the caller's expected version so the
// storage layer's WHERE version predicate enforces the caller's precondition.
// It returns the persisted category with the new version and updated_at.
func (s *Service) UpdateCategory(ctx context.Context, req UpdateCategoryRequest) (domain.Category, error) {
	if req.Version <= 0 {
		s.m.IncCategoryUpdated(metrics.StatusError)
		return domain.Category{}, domain.ErrInvalidVersion
	}
	c, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		s.m.IncCategoryUpdated(metrics.StatusError)
		return domain.Category{}, err
	}
	if err = c.UpdateNameAndIcon(req.Name, req.Icon); err != nil {
		s.m.IncCategoryUpdated(metrics.StatusError)
		return domain.Category{}, err
	}
	c.Version = req.Version
	persisted, err := s.repo.Update(ctx, c)
	if err != nil {
		s.m.IncCategoryUpdated(metrics.StatusError)
		return domain.Category{}, err
	}
	s.m.IncCategoryUpdated(metrics.StatusSuccess)
	return persisted, nil
}
