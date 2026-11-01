package application

import (
	"context"
	"minimart/internal/address/domain"
	"minimart/pkg/metrics"
)

type UpdateAddressRequest struct {
	ID         string
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
	Version    int
}

func (s *Service) UpdateAddress(ctx context.Context, req UpdateAddressRequest) (domain.Address, error) {
	if req.CustomerID == "" {
		s.m.IncAddressUpdated(metrics.StatusError)
		return domain.Address{}, domain.ErrEmptyCustomerID
	}
	if req.Version <= 0 {
		s.m.IncAddressUpdated(metrics.StatusError)
		return domain.Address{}, domain.ErrInvalidVersion
	}
	a, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		s.m.IncAddressUpdated(metrics.StatusError)
		return domain.Address{}, err
	}
	if a.CustomerID != req.CustomerID {
		s.m.IncAddressUpdated(metrics.StatusError)
		return domain.Address{}, domain.ErrAddressNotFound
	}
	if err = a.UpdateDetails(domain.AddressFields{
		Label:    req.Label,
		FullName: req.FullName,
		Street:   req.Street,
		City:     req.City,
		State:    req.State,
		Zip:      req.Zip,
		Country:  req.Country,
		Phone:    req.Phone,
	}); err != nil {
		s.m.IncAddressUpdated(metrics.StatusError)
		return domain.Address{}, err
	}
	a.Version = req.Version
	persisted, err := s.repo.Update(ctx, a)
	if err != nil {
		s.m.IncAddressUpdated(metrics.StatusError)
		return domain.Address{}, err
	}

	// If IsDefault is true and address wasn't already default, set it as default
	if req.IsDefault && !persisted.IsDefault {
		persisted, err = s.repo.SetDefault(ctx, persisted.ID, persisted.CustomerID)
		if err != nil {
			s.m.IncAddressUpdated(metrics.StatusError)
			return domain.Address{}, err
		}
	}

	s.m.IncAddressUpdated(metrics.StatusSuccess)
	return persisted, nil
}
