package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

func (s *Service) ProcessShipment(ctx context.Context, id string) (domain.Shipment, error) {
	shipment, err := s.repo.GetShipment(ctx, id)
	if err != nil {
		s.m.IncShipmentProcess(metrics.StatusError)
		return domain.Shipment{}, err
	}
	if err := shipment.Process(); err != nil {
		s.m.IncShipmentProcess(metrics.StatusError)
		return domain.Shipment{}, err
	}
	if err := s.repo.UpdateShipment(ctx, shipment); err != nil {
		s.m.IncShipmentProcess(metrics.StatusError)
		return domain.Shipment{}, err
	}
	s.m.IncShipmentProcess(metrics.StatusSuccess)
	return shipment, nil
}
