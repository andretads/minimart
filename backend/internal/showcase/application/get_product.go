package application

import (
	"context"
	"minimart/internal/showcase/domain"
)

// GetProduct returns a single product by ID with all of its images.
// Returns domain.ErrProductNotFound when no product has the given ID.
func (s *Service) GetProduct(ctx context.Context, id string) (domain.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Product{}, err
	}
	images, err := s.repo.ListImages(ctx, product.ID)
	if err != nil {
		return domain.Product{}, err
	}
	product.Images = images
	return product, nil
}
