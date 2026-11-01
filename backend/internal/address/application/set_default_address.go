package application

import (
	"context"
	"minimart/internal/address/domain"
	"minimart/pkg/metrics"
)

type SetDefaultAddressRequest struct {
	ID         string
	CustomerID string
}

func (s *Service) SetDefaultAddress(ctx context.Context, req SetDefaultAddressRequest) (domain.Address, error) {
	if req.CustomerID == "" {
		s.m.IncAddressDefaultSet(metrics.StatusError)
		return domain.Address{}, domain.ErrEmptyCustomerID
	}
	a, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		s.m.IncAddressDefaultSet(metrics.StatusError)
		return domain.Address{}, err
	}
	if a.CustomerID != req.CustomerID {
		s.m.IncAddressDefaultSet(metrics.StatusError)
		return domain.Address{}, domain.ErrAddressNotFound
	}
	persisted, err := s.repo.SetDefault(ctx, req.ID, req.CustomerID)
	if err != nil {
		s.m.IncAddressDefaultSet(metrics.StatusError)
		return domain.Address{}, err
	}
	s.m.IncAddressDefaultSet(metrics.StatusSuccess)
	return persisted, nil
}
