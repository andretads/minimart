package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

type UpdateZoneRequest struct {
	Name            string
	Country         string
	State           string
	City            string
	PostalCodeStart string
	PostalCodeEnd   string
}

func (s *Service) UpdateZone(ctx context.Context, id string, req UpdateZoneRequest) (domain.ShippingZone, error) {
	zone, err := s.repo.GetZone(ctx, id)
	if err != nil {
		s.m.IncShippingZoneUpdate(metrics.StatusError)
		return domain.ShippingZone{}, err
	}
	if err := zone.Update(req.Name, req.Country, req.State, req.City, req.PostalCodeStart, req.PostalCodeEnd); err != nil {
		s.m.IncShippingZoneUpdate(metrics.StatusError)
		return domain.ShippingZone{}, err
	}
	if err := s.repo.UpdateZone(ctx, zone); err != nil {
		s.m.IncShippingZoneUpdate(metrics.StatusError)
		return domain.ShippingZone{}, err
	}
	s.m.IncShippingZoneUpdate(metrics.StatusSuccess)
	return zone, nil
}
