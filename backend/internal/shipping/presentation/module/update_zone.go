package module

import (
	"context"
	"errors"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// UpdateZone updates an existing shipping zone.
func (s *Shipping) UpdateZone(ctx context.Context, id string, req shippingclient.UpdateZoneRequest) (shippingclient.ShippingZone, error) {
	z, err := s.svc.UpdateZone(ctx, id, application.UpdateZoneRequest{
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
