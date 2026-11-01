package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

func (s *Service) GetZone(ctx context.Context, id string) (domain.ShippingZone, error) {
	zone, err := s.repo.GetZone(ctx, id)
	if err != nil {
		s.m.IncShippingZoneGet(metrics.StatusError)
		return domain.ShippingZone{}, err
	}
	s.m.IncShippingZoneGet(metrics.StatusSuccess)
	return zone, nil
}
