package application

import (
	"context"
	"minimart/internal/address/domain"
	"minimart/pkg/metrics"
)

func (s *Service) GetAddress(ctx context.Context, id, customerID string) (domain.Address, error) {
	if customerID == "" {
		s.m.IncAddressFetched(metrics.StatusError)
		return domain.Address{}, domain.ErrEmptyCustomerID
	}
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.m.IncAddressFetched(metrics.StatusError)
		return domain.Address{}, err
	}
	if a.CustomerID != customerID {
		s.m.IncAddressFetched(metrics.StatusError)
		return domain.Address{}, domain.ErrAddressNotFound
	}
	s.m.IncAddressFetched(metrics.StatusSuccess)
	return a, nil
}
