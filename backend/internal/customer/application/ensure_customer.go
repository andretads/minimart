package application

import (
	"context"
	"errors"
	"minimart/internal/customer/domain"
	"minimart/pkg/metrics"
)

// EnsureCustomer resolves a user to their customer, creating one if missing.
// It is the get-or-create entry point used internally by other domains (e.g.
// order placement) and stays idempotent under concurrent callers. It first
// attempts a lookup by user_id; on not-found, it creates a new customer with
// an empty display name (to be filled later by the user) and saves it,
// handling the race where another caller creates the same customer concurrently.
func (s *Service) EnsureCustomer(ctx context.Context, userID string) (domain.Customer, error) {
	if c, err := s.repo.GetByUserID(ctx, userID); err == nil {
		s.m.IncCustomerEnsured(metrics.StatusSuccess)
		return c, nil
	} else if !errors.Is(err, domain.ErrCustomerNotFound) {
		s.m.IncCustomerEnsured(metrics.StatusError)
		return domain.Customer{}, err
	}
	c, err := domain.NewCustomer(newID(), userID, "")
	if err != nil {
		s.m.IncCustomerEnsured(metrics.StatusError)
		return domain.Customer{}, err
	}
	if err := s.repo.Save(ctx, c); err != nil {
		if errors.Is(err, domain.ErrCustomerExists) {
			if c2, err2 := s.repo.GetByUserID(ctx, userID); err2 == nil {
				s.m.IncCustomerEnsured(metrics.StatusSuccess)
				return c2, nil
			}
		}
		s.m.IncCustomerEnsured(metrics.StatusError)
		return domain.Customer{}, err
	}
	s.m.IncCustomerEnsured(metrics.StatusSuccess)
	return c, nil
}
