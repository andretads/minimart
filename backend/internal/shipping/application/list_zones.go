package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

func (s *Service) ListZones(ctx context.Context) ([]domain.ShippingZone, error) {
	zones, err := s.repo.ListZones(ctx)
	if err != nil {
		s.m.IncShippingZonesList(metrics.StatusError)
		return nil, err
	}
	s.m.IncShippingZonesList(metrics.StatusSuccess)
	return zones, nil
}
