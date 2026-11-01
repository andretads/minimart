package application

import (
	"context"
	"minimart/internal/address/domain"
	"minimart/pkg/metrics"
)

func (s *Service) GetDefaultAddress(ctx context.Context, customerID string) (domain.Address, error) {
	if customerID == "" {
		s.m.IncAddressFetched(metrics.StatusError)
		return domain.Address{}, domain.ErrEmptyCustomerID
	}
	a, err := s.repo.GetDefaultByCustomerID(ctx, customerID)
	if err != nil {
		s.m.IncAddressFetched(metrics.StatusError)
		return domain.Address{}, err
	}
	s.m.IncAddressFetched(metrics.StatusSuccess)
	return a, nil
}
