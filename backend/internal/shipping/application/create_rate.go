package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

type CreateRateRequest struct {
	ZoneID             string
	MethodID           string
	BaseCostCents      int64
	FreeThresholdCents int64
	Active             bool
}

func (s *Service) CreateRate(ctx context.Context, req CreateRateRequest) (domain.ShippingRate, error) {
	rate, err := domain.NewShippingRate(req.ZoneID, req.MethodID, req.BaseCostCents, req.FreeThresholdCents, req.Active)
	if err != nil {
		s.m.IncShippingRateCreate(metrics.StatusError)
		return domain.ShippingRate{}, err
	}
	if err := s.repo.CreateRate(ctx, rate); err != nil {
		s.m.IncShippingRateCreate(metrics.StatusError)
		return domain.ShippingRate{}, err
	}
	s.m.IncShippingRateCreate(metrics.StatusSuccess)
	return rate, nil
}
