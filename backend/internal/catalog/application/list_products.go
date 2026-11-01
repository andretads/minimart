package application

import (
	"context"
	"minimart/internal/catalog/domain"
)

// ListProducts returns the whole catalog.
func (s *Service) ListProducts(ctx context.Context) ([]domain.Product, error) {
	return s.repo.List(ctx)
}

// ListProductsByCategory returns all products in the given category.
func (s *Service) ListProductsByCategory(ctx context.Context, categoryID string) ([]domain.Product, error) {
	return s.repo.ListByCategory(ctx, categoryID)
}
