package application

import (
	"context"
	"minimart/internal/address/domain"
	"minimart/pkg/metrics"
)

type DeleteAddressRequest struct {
	ID         string
	CustomerID string
}

func (s *Service) DeleteAddress(ctx context.Context, req DeleteAddressRequest) error {
	if req.CustomerID == "" {
		s.m.IncAddressDeleted(metrics.StatusError)
		return domain.ErrEmptyCustomerID
	}
	a, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		s.m.IncAddressDeleted(metrics.StatusError)
		return err
	}
	if a.CustomerID != req.CustomerID {
		s.m.IncAddressDeleted(metrics.StatusError)
		return domain.ErrAddressNotFound
	}
	if err := s.repo.Delete(ctx, req.ID, req.CustomerID); err != nil {
		s.m.IncAddressDeleted(metrics.StatusError)
		return err
	}
	s.m.IncAddressDeleted(metrics.StatusSuccess)
	return nil
}
