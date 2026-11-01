package application

import (
	"context"
	"errors"
	"minimart/internal/category/domain"
	"minimart/pkg/metrics"
)

// CreateCategoryRequest carries the input for creating a category.
type CreateCategoryRequest struct {
	Name string
	Icon string
}

// CreateCategory creates a category record and assigns it a unique ID.
// It validates the input, constructs the domain entity, persists it, and records metrics.
// Returns ErrCategoryExists if a category with the same name (case-insensitive) already exists.
func (s *Service) CreateCategory(ctx context.Context, req CreateCategoryRequest) (domain.Category, error) {
	c, err := domain.NewCategory(newID(), req.Name, req.Icon)
	if err != nil {
		s.m.IncCategoryCreated(metrics.StatusError)
		return domain.Category{}, err
	}
	if err := s.repo.Save(ctx, c); err != nil {
		if errors.Is(err, domain.ErrCategoryExists) {
			s.m.IncCategoryCreated(metrics.StatusError)
			return domain.Category{}, domain.ErrCategoryExists
		}
		s.m.IncCategoryCreated(metrics.StatusError)
		return domain.Category{}, err
	}
	s.m.IncCategoryCreated(metrics.StatusSuccess)
	return c, nil
}
