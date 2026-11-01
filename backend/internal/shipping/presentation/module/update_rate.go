package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// UpdateRate updates an existing shipping rate.
func (s *Shipping) UpdateRate(ctx context.Context, id string, req shippingclient.UpdateRateRequest) (shippingclient.ShippingRate, error) {
	r, err := s.svc.UpdateRate(ctx, id, application.UpdateRateRequest{
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
