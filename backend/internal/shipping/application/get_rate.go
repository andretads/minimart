package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

func (s *Service) GetRate(ctx context.Context, id string) (domain.ShippingRate, error) {
	rate, err := s.repo.GetRate(ctx, id)
	if err != nil {
		s.m.IncShippingRateGet(metrics.StatusError)
		return domain.ShippingRate{}, err
	}
	s.m.IncShippingRateGet(metrics.StatusSuccess)
	return rate, nil
}
