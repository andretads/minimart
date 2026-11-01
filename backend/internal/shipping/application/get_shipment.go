package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

func (s *Service) GetShipment(ctx context.Context, id string) (domain.Shipment, error) {
	shipment, err := s.repo.GetShipment(ctx, id)
	if err != nil {
		s.m.IncShipmentGet(metrics.StatusError)
		return domain.Shipment{}, err
	}
	s.m.IncShipmentGet(metrics.StatusSuccess)
	return shipment, nil
}

func (s *Service) GetShipmentByOrderID(ctx context.Context, orderID string) (domain.Shipment, error) {
	shipment, err := s.repo.GetShipmentByOrderID(ctx, orderID)
	if err != nil {
		s.m.IncShipmentGet(metrics.StatusError)
		return domain.Shipment{}, err
	}
	s.m.IncShipmentGet(metrics.StatusSuccess)
	return shipment, nil
}
