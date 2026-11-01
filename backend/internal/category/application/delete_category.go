package application

import (
	"context"
	"minimart/internal/category/domain"
	"minimart/pkg/metrics"
)

// DeleteCategoryRequest carries the input for deleting a category.
// Version is the optimistic lock token the caller read from a prior GET; it must be positive.
type DeleteCategoryRequest struct {
	ID      string
	Version int
}

// DeleteCategory deletes a category by ID under optimistic lock.
// It validates the version, then delegates to the repository.
func (s *Service) DeleteCategory(ctx context.Context, req DeleteCategoryRequest) error {
	if req.Version <= 0 {
		s.m.IncCategoryDeleted(metrics.StatusError)
		return domain.ErrInvalidVersion
	}
	if err := s.repo.Delete(ctx, req.ID, req.Version); err != nil {
		s.m.IncCategoryDeleted(metrics.StatusError)
		return err
	}
	s.m.IncCategoryDeleted(metrics.StatusSuccess)
	return nil
}
