package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// CreateRate creates a new shipping rate.
func (s *Shipping) CreateRate(ctx context.Context, req shippingclient.CreateRateRequest) (shippingclient.ShippingRate, error) {
	r, err := s.svc.CreateRate(ctx, application.CreateRateRequest{
		ZoneID:             req.ZoneID,
		MethodID:           req.MethodID,
		BaseCostCents:      req.BaseCostCents,
		FreeThresholdCents: req.FreeThresholdCents,
		Active:             req.Active,
	})
	if err != nil {
		if errors.Is(err, domain.ErrRateNotFound) {
			return shippingclient.ShippingRate{}, shippingclient.ErrRateNotFound
		}
		return shippingclient.ShippingRate{}, err
	}
	return toClientShippingRate(r), nil
}
