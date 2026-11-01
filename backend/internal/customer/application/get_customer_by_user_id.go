package application

import (
	"context"
	"minimart/internal/customer/domain"
)

// GetCustomerByUserID resolves the authenticated user's UserID to their
// CustomerID. Returns ErrCustomerNotFound when no customer exists yet;
// callers that want to create one should use EnsureCustomer instead.
func (s *Service) GetCustomerByUserID(ctx context.Context, userID string) (domain.Customer, error) {
	return s.repo.GetByUserID(ctx, userID)
}
