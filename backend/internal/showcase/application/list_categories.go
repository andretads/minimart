package application

import (
	"context"
	"minimart/internal/showcase/domain"
)

// ListCategories returns all categories for public display.
func (s *Service) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return s.repo.ListCategories(ctx)
}
