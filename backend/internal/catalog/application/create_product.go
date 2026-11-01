package application

import (
	"context"
	"minimart/internal/catalog/domain"
	"minimart/pkg/metrics"
)

// CreateProductRequest is the input to the CreateProduct use case.
type CreateProductRequest struct {
	Name        string
	PriceCents  int64
	Stock       int
	CategoryID  string
	SKU         string
	Description string
	Details     []domain.ProductDetail
}

// CreateProduct mints an ID, builds a valid Product, and saves it through the port.
func (s *Service) CreateProduct(ctx context.Context, req CreateProductRequest) (domain.Product, error) {
	// Validate category exists
	_, err := s.categoryProvider.GetCategory(ctx, req.CategoryID)
	if err != nil {
		s.m.IncProductCreated(metrics.StatusError)
		return domain.Product{}, err
	}

	p, err := domain.NewProduct(newID(), req.Name, req.PriceCents, req.Stock, req.CategoryID, req.SKU, req.Description, req.Details)
	if err != nil {
		s.m.IncProductCreated(metrics.StatusError)
		return domain.Product{}, err
	}

	if err := s.repo.Save(ctx, p); err != nil {
		s.m.IncProductCreated(metrics.StatusError)
		return domain.Product{}, err
	}
	s.m.IncProductCreated(metrics.StatusSuccess)
	return p, nil
}
