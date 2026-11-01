package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/metrics"
)

func (s *Service) GetTrackingEvents(ctx context.Context, shipmentID string) ([]domain.TrackingEvent, error) {
	events, err := s.repo.GetTrackingEvents(ctx, shipmentID)
	if err != nil {
		s.m.IncTrackingEventList(metrics.StatusError)
		return nil, err
	}
	s.m.IncTrackingEventList(metrics.StatusSuccess)
	return events, nil
}
