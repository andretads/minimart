package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// UpdateMethod updates an existing shipping method.
func (s *Shipping) UpdateMethod(ctx context.Context, id string, req shippingclient.UpdateMethodRequest) (shippingclient.ShippingMethod, error) {
	m, err := s.svc.UpdateMethod(ctx, id, application.UpdateMethodRequest{
		Name:             req.Name,
		Description:      req.Description,
		Active:           req.Active,
		EstimatedMinDays: req.EstimatedMinDays,
		EstimatedMaxDays: req.EstimatedMaxDays,
	})
	if err != nil {
		if errors.Is(err, domain.ErrMethodNotFound) {
			return shippingclient.ShippingMethod{}, shippingclient.ErrMethodNotFound
		}
		return shippingclient.ShippingMethod{}, err
	}
	return toClientShippingMethod(m), nil
}
