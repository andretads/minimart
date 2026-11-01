package application

import (
	"context"
	"errors"
	"minimart/internal/address/domain"
	"minimart/pkg/metrics"
)

type CreateAddressRequest struct {
	CustomerID string
	Label      string
	FullName   string
	Street     string
	City       string
	State      string
	Zip        string
	Country    string
	Phone      string
	IsDefault  bool
}

func (s *Service) CreateAddress(ctx context.Context, req CreateAddressRequest) (domain.Address, error) {
	if req.CustomerID == "" {
		s.m.IncAddressCreated(metrics.StatusError)
		return domain.Address{}, domain.ErrEmptyCustomerID
	}
	a, err := domain.NewAddress(newID(), req.CustomerID, domain.AddressFields{
		Label:    req.Label,
		FullName: req.FullName,
		Street:   req.Street,
		City:     req.City,
		State:    req.State,
		Zip:      req.Zip,
		Country:  req.Country,
		Phone:    req.Phone,
	}, req.IsDefault)
	if err != nil {
		s.m.IncAddressCreated(metrics.StatusError)
		return domain.Address{}, err
	}
	if err := s.repo.Save(ctx, a); err != nil {
		if errors.Is(err, domain.ErrAddressExists) {
			s.m.IncAddressCreated(metrics.StatusError)
			return domain.Address{}, domain.ErrAddressExists
		}
		s.m.IncAddressCreated(metrics.StatusError)
		return domain.Address{}, err
	}
	s.m.IncAddressCreated(metrics.StatusSuccess)
	return a, nil
}
