package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
	"time"
)

type AddTrackingEventRequest struct {
	ShipmentID  string
	Status      domain.ShipmentStatus
	Description string
	Location    string
	OccurredAt  time.Time
}

func (s *Service) AddTrackingEvent(ctx context.Context, req AddTrackingEventRequest) (domain.TrackingEvent, error) {
	// Verify shipment exists
	_, err := s.repo.GetShipment(ctx, req.ShipmentID)
	if err != nil {
		s.m.IncTrackingEventCreate(metrics.StatusError)
		return domain.TrackingEvent{}, err
	}

	event, err := domain.NewTrackingEvent(req.ShipmentID, req.Status, req.Description, req.Location, req.OccurredAt)
	if err != nil {
		s.m.IncTrackingEventCreate(metrics.StatusError)
		return domain.TrackingEvent{}, err
	}
	if err := s.repo.AddTrackingEvent(ctx, event); err != nil {
		s.m.IncTrackingEventCreate(metrics.StatusError)
		return domain.TrackingEvent{}, err
	}
	s.m.IncTrackingEventCreate(metrics.StatusSuccess)
	return event, nil
}
