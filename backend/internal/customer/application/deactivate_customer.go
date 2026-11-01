package application

import (
	"context"
	"minimart/internal/customer/domain"
	"minimart/pkg/metrics"
)

// DeactivateCustomer prevents a customer from performing new operations that
// require an active customer. It does not delete historical orders or
// payments, and is idempotent: deactivating an already-deactivated customer
// returns it unchanged (the storage layer bumps the version on every call,
// mirroring catalog's stock reserve/restore pattern). It loads the customer,
// applies the domain-level Deactivate (which is idempotent and preserves the
// original deactivated_at), persists the change, and records metrics.
func (s *Service) DeactivateCustomer(ctx context.Context, id string) (domain.Customer, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Customer{}, err
	}
	// Domain Deactivate is idempotent; we pass the entity to the repo which
	// unconditionally bumps the version (COALESCE preserves deactivated_at).
	persisted, err := s.repo.Deactivate(ctx, c)
	if err != nil {
		return domain.Customer{}, err
	}
	s.m.IncCustomerDeactivated(metrics.StatusSuccess)
	return persisted, nil
}
