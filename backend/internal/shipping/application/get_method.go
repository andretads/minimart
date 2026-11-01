package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

func (s *Service) GetMethod(ctx context.Context, id string) (domain.ShippingMethod, error) {
	method, err := s.repo.GetMethod(ctx, id)
	if err != nil {
		s.m.IncShippingMethodGet(metrics.StatusError)
		return domain.ShippingMethod{}, err
	}
	s.m.IncShippingMethodGet(metrics.StatusSuccess)
	return method, nil
}
