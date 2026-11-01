package application

import (
	"context"
	"minimart/internal/category/domain"
)

// ListCategories returns all categories.
func (s *Service) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return s.repo.List(ctx)
}
