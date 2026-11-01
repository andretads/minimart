package application

import (
	"context"
	"errors"
	"minimart/internal/customer/domain"
	"minimart/pkg/metrics"
)

// CreateCustomerRequest carries the input for creating a customer record.
type CreateCustomerRequest struct {
	UserID      string
	DisplayName string
}

// CreateCustomer creates a customer record and assigns it a unique CustomerID,
// linked to userID when applicable (userID is required). It validates the
// input, constructs the domain entity, persists it, and records metrics.
// Returns ErrCustomerExists if a customer with the same user_id already exists.
func (s *Service) CreateCustomer(ctx context.Context, req CreateCustomerRequest) (domain.Customer, error) {
	if req.UserID == "" {
		s.m.IncCustomerCreated(metrics.StatusError)
		return domain.Customer{}, domain.ErrEmptyUserID
	}
	c, err := domain.NewCustomer(newID(), req.UserID, req.DisplayName)
	if err != nil {
		s.m.IncCustomerCreated(metrics.StatusError)
		return domain.Customer{}, err
	}
	if err := s.repo.Save(ctx, c); err != nil {
		if errors.Is(err, domain.ErrCustomerExists) {
			s.m.IncCustomerCreated(metrics.StatusError)
			return domain.Customer{}, domain.ErrCustomerExists
		}
		s.m.IncCustomerCreated(metrics.StatusError)
		return domain.Customer{}, err
	}
	s.m.IncCustomerCreated(metrics.StatusSuccess)
	return c, nil
}
