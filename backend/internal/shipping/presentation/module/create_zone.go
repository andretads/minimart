package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// CreateZone creates a new shipping zone.
func (s *Shipping) CreateZone(ctx context.Context, req shippingclient.CreateZoneRequest) (shippingclient.ShippingZone, error) {
	z, err := s.svc.CreateZone(ctx, application.CreateZoneRequest{
		Name:            req.Name,
		Country:         req.Country,
		State:           req.State,
		City:            req.City,
		PostalCodeStart: req.PostalCodeStart,
		PostalCodeEnd:   req.PostalCodeEnd,
	})
	if err != nil {
		if errors.Is(err, domain.ErrZoneNotFound) {
			return shippingclient.ShippingZone{}, shippingclient.ErrZoneNotFound
		}
		return shippingclient.ShippingZone{}, err
	}
	return toClientShippingZone(z), nil
}
