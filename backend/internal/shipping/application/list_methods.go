package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

func (s *Service) ListMethods(ctx context.Context) ([]domain.ShippingMethod, error) {
	methods, err := s.repo.ListActiveMethods(ctx)
	if err != nil {
		s.m.IncShippingMethodsList(metrics.StatusError)
		return nil, err
	}
	s.m.IncShippingMethodsList(metrics.StatusSuccess)
	return methods, nil
}
