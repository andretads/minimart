package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// CreateMethod creates a new shipping method.
func (s *Shipping) CreateMethod(ctx context.Context, req shippingclient.CreateMethodRequest) (shippingclient.ShippingMethod, error) {
	m, err := s.svc.CreateMethod(ctx, application.CreateMethodRequest{
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
