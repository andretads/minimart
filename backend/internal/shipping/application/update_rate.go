package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

type UpdateRateRequest struct {
	BaseCostCents      int64
	FreeThresholdCents int64
	Active             bool
}

func (s *Service) UpdateRate(ctx context.Context, id string, req UpdateRateRequest) (domain.ShippingRate, error) {
	rate, err := s.repo.GetRate(ctx, id)
	if err != nil {
		s.m.IncShippingRateUpdate(metrics.StatusError)
		return domain.ShippingRate{}, err
	}
	if err := rate.Update(req.BaseCostCents, req.FreeThresholdCents, req.Active); err != nil {
		s.m.IncShippingRateUpdate(metrics.StatusError)
		return domain.ShippingRate{}, err
	}
	if err := s.repo.UpdateRate(ctx, rate); err != nil {
		s.m.IncShippingRateUpdate(metrics.StatusError)
		return domain.ShippingRate{}, err
	}
	s.m.IncShippingRateUpdate(metrics.StatusSuccess)
	return rate, nil
}
