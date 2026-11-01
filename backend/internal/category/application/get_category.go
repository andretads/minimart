package application

import (
	"context"
	"minimart/internal/category/domain"
)

// GetCategory retrieves a category by ID.
func (s *Service) GetCategory(ctx context.Context, id string) (domain.Category, error) {
	return s.repo.GetByID(ctx, id)
}
