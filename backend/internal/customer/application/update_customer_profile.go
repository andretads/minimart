package application

import (
	"context"
	"minimart/internal/customer/domain"
	"minimart/pkg/metrics"
)

// UpdateCustomerProfileRequest carries the input for updating a customer's profile.
// Version is the optimistic lock token the caller read from a prior GET; it must
// be positive. The request uses the same pattern as catalog's UpdateProductRequest.
type UpdateCustomerProfileRequest struct {
	ID          string
	DisplayName string
	Version     int
}

// UpdateCustomerProfile updates customer-owned profile fields such as the
// display name. It loads the customer, applies the domain-level validation via
// UpdateDisplayName, then overwrites the loaded entity's Version with the
// caller's expected version so the storage layer's WHERE version predicate
// enforces the caller's precondition. It returns the persisted customer with
// the new version and updated_at.
func (s *Service) UpdateCustomerProfile(ctx context.Context, req UpdateCustomerProfileRequest) (domain.Customer, error) {
	if req.Version <= 0 {
		s.m.IncCustomerUpdated(metrics.StatusError)
		return domain.Customer{}, domain.ErrInvalidVersion
	}
	c, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		s.m.IncCustomerUpdated(metrics.StatusError)
		return domain.Customer{}, err
	}
	if err = c.UpdateDisplayName(req.DisplayName); err != nil {
		s.m.IncCustomerUpdated(metrics.StatusError)
		return domain.Customer{}, err
	}
	// Carry the caller's expected version onto the write so the storage layer's
	// WHERE version predicate enforces the caller's precondition, not ours.
	c.Version = req.Version
	persisted, err := s.repo.UpdateProfile(ctx, c)
	if err != nil {
		s.m.IncCustomerUpdated(metrics.StatusError)
		return domain.Customer{}, err
	}
	s.m.IncCustomerUpdated(metrics.StatusSuccess)
	return persisted, nil
}
