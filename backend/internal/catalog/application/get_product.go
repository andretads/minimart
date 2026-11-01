package application

import (
	"context"
	"minimart/internal/catalog/domain"
)

// GetProduct returns a product by ID, or domain.ErrProductNotFound.
func (s *Service) GetProduct(ctx context.Context, id string) (domain.Product, error) {
	return s.repo.GetByID(ctx, id)
}
