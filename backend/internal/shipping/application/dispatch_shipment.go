package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

type DispatchShipmentRequest struct {
	TrackingCode string
	Carrier      string
}

// DispatchShipment moves a processing shipment to shipped. The repository
// persists the new state and the shipment-shipped event in one transaction, so
// the event is published at least once by the outbox relay.
func (s *Service) DispatchShipment(ctx context.Context, id string, req DispatchShipmentRequest) (domain.Shipment, error) {
	shipment, err := s.repo.GetShipment(ctx, id)
	if err != nil {
		s.m.IncShipmentDispatch(metrics.StatusError)
		return domain.Shipment{}, err
	}
	if err := shipment.Dispatch(req.TrackingCode, req.Carrier); err != nil {
		s.m.IncShipmentDispatch(metrics.StatusError)
		return domain.Shipment{}, err
	}
	if err := s.repo.UpdateShipment(ctx, shipment); err != nil {
		s.m.IncShipmentDispatch(metrics.StatusError)
		return domain.Shipment{}, err
	}

	s.m.IncShipmentDispatch(metrics.StatusSuccess)
	return shipment, nil
}
