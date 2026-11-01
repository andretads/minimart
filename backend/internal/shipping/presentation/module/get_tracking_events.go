package module

import (
	"context"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// GetTrackingEvents returns tracking events for a shipment.
func (s *Shipping) GetTrackingEvents(ctx context.Context, shipmentID string) ([]shippingclient.TrackingEvent, error) {
	events, err := s.svc.GetTrackingEvents(ctx, shipmentID)
	if err != nil {
		return nil, err
	}
	resp := make([]shippingclient.TrackingEvent, len(events))
	for i, e := range events {
		resp[i] = toClientTrackingEvent(e)
	}
	return resp, nil
}
