package application

import (
	"context"
	"minimart/internal/customer/domain"
)

// GetCustomerByID retrieves customer information using CustomerID.
// It is the primary lookup for other domains that need customer information
// (e.g., order placement resolving a customer reference).
func (s *Service) GetCustomerByID(ctx context.Context, id string) (domain.Customer, error) {
	return s.repo.GetByID(ctx, id)
}
