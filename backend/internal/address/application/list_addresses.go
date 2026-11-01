package application

import (
	"context"
	"minimart/internal/address/domain"
	"minimart/pkg/metrics"
)

type ListAddressesRequest struct {
	CustomerID string
}

func (s *Service) ListAddresses(ctx context.Context, req ListAddressesRequest) ([]domain.Address, error) {
	if req.CustomerID == "" {
		s.m.IncAddressListed(metrics.StatusError)
		return nil, domain.ErrEmptyCustomerID
	}
	addrs, err := s.repo.GetByCustomerID(ctx, req.CustomerID)
	if err != nil {
		s.m.IncAddressListed(metrics.StatusError)
		return nil, err
	}
	s.m.IncAddressListed(metrics.StatusSuccess)
	return addrs, nil
}
