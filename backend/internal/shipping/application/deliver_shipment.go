package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

// DeliverShipment moves a shipped shipment to delivered. The repository
// persists the new state and the shipment-delivered event in one transaction,
// so the event is published at least once by the outbox relay.
func (s *Service) DeliverShipment(ctx context.Context, id string) (domain.Shipment, error) {
	shipment, err := s.repo.GetShipment(ctx, id)
	if err != nil {
		s.m.IncShipmentDeliver(metrics.StatusError)
		return domain.Shipment{}, err
	}
	if err := shipment.Deliver(); err != nil {
		s.m.IncShipmentDeliver(metrics.StatusError)
		return domain.Shipment{}, err
	}
	if err := s.repo.UpdateShipment(ctx, shipment); err != nil {
		s.m.IncShipmentDeliver(metrics.StatusError)
		return domain.Shipment{}, err
	}

	s.m.IncShipmentDeliver(metrics.StatusSuccess)
	return shipment, nil
}
