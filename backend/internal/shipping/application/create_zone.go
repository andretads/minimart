package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

type CreateZoneRequest struct {
	Name            string
	Country         string
	State           string
	City            string
	PostalCodeStart string
	PostalCodeEnd   string
}

func (s *Service) CreateZone(ctx context.Context, req CreateZoneRequest) (domain.ShippingZone, error) {
	zone, err := domain.NewShippingZone(req.Name, req.Country, req.State, req.City, req.PostalCodeStart, req.PostalCodeEnd)
	if err != nil {
		s.m.IncShippingZoneCreate(metrics.StatusError)
		return domain.ShippingZone{}, err
	}
	if err := s.repo.CreateZone(ctx, zone); err != nil {
		s.m.IncShippingZoneCreate(metrics.StatusError)
		return domain.ShippingZone{}, err
	}
	s.m.IncShippingZoneCreate(metrics.StatusSuccess)
	return zone, nil
}
