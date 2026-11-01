package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

func (s *Service) ListRates(ctx context.Context, zoneID, methodID string) ([]domain.ShippingRate, error) {
	rates, err := s.repo.ListRates(ctx, zoneID, methodID)
	if err != nil {
		s.m.IncShippingRatesList(metrics.StatusError)
		return nil, err
	}
	s.m.IncShippingRatesList(metrics.StatusSuccess)
	return rates, nil
}
