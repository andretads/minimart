package domain

import (
	"time"

	"github.com/google/uuid"
)

type TrackingEvent struct {
	ID          string
	ShipmentID  string
	Status      ShipmentStatus
	Description string
	Location    string
	OccurredAt  time.Time
	CreatedAt   time.Time
}

func NewTrackingEvent(shipmentID string, status ShipmentStatus, description, location string, occurredAt time.Time) (TrackingEvent, error) {
	if shipmentID == "" {
		return TrackingEvent{}, ErrShipmentNotFound
	}
	if description == "" {
		return TrackingEvent{}, ErrInvalidTransition
	}
	return TrackingEvent{
		ID:          uuid.New().String(),
		ShipmentID:  shipmentID,
		Status:      status,
		Description: description,
		Location:    location,
		OccurredAt:  occurredAt.UTC(),
		CreatedAt:   time.Now().UTC(),
	}, nil
}
