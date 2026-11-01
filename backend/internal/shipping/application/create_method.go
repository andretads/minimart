package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

type CreateMethodRequest struct {
	Name             string
	Description      string
	Active           bool
	EstimatedMinDays int
	EstimatedMaxDays int
}

func (s *Service) CreateMethod(ctx context.Context, req CreateMethodRequest) (domain.ShippingMethod, error) {
	method, err := domain.NewShippingMethod(req.Name, req.Description, req.Active, req.EstimatedMinDays, req.EstimatedMaxDays)
	if err != nil {
		s.m.IncShippingMethodCreate(metrics.StatusError)
		return domain.ShippingMethod{}, err
	}
	if err := s.repo.CreateMethod(ctx, method); err != nil {
		s.m.IncShippingMethodCreate(metrics.StatusError)
		return domain.ShippingMethod{}, err
	}
	s.m.IncShippingMethodCreate(metrics.StatusSuccess)
	return method, nil
}
