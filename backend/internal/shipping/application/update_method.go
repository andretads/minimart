package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

type UpdateMethodRequest struct {
	Name             string
	Description      string
	Active           bool
	EstimatedMinDays int
	EstimatedMaxDays int
}

func (s *Service) UpdateMethod(ctx context.Context, id string, req UpdateMethodRequest) (domain.ShippingMethod, error) {
	method, err := s.repo.GetMethod(ctx, id)
	if err != nil {
		s.m.IncShippingMethodUpdate(metrics.StatusError)
		return domain.ShippingMethod{}, err
	}
	if err := method.Update(req.Name, req.Description, req.Active, req.EstimatedMinDays, req.EstimatedMaxDays); err != nil {
		s.m.IncShippingMethodUpdate(metrics.StatusError)
		return domain.ShippingMethod{}, err
	}
	if err := s.repo.UpdateMethod(ctx, method); err != nil {
		s.m.IncShippingMethodUpdate(metrics.StatusError)
		return domain.ShippingMethod{}, err
	}
	s.m.IncShippingMethodUpdate(metrics.StatusSuccess)
	return method, nil
}
