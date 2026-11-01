package domain

import (
	"time"

	"github.com/google/uuid"
)

type ShipmentStatus string

const (
	ShipmentStatusPending    ShipmentStatus = "pending"
	ShipmentStatusProcessing ShipmentStatus = "processing"
	ShipmentStatusShipped    ShipmentStatus = "shipped"
	ShipmentStatusDelivered  ShipmentStatus = "delivered"
	ShipmentStatusFailed     ShipmentStatus = "failed"
)

type Shipment struct {
	ID                  string
	OrderID             string
	UserID              string
	MethodID            string
	TrackingCode        string
	Carrier             string
	ShippingCostCents   int64
	DestinationSnapshot DestinationSnapshot
	Status              ShipmentStatus
	ShippedAt           *time.Time
	DeliveredAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time

	// events are the domain events raised since the shipment was created or
	// loaded. The repository persists them to the outbox in the same
	// transaction as the shipment, then they are discarded with the value.
	events []Event
}

// Events returns the domain events raised since the shipment was created or
// loaded.
func (s Shipment) Events() []Event { return s.events }

func NewShipment(orderID, userID, methodID string, destination DestinationSnapshot, shippingCostCents int64) (Shipment, error) {
	if orderID == "" || userID == "" || methodID == "" {
		return Shipment{}, ErrInvalidInput
	}
	if err := validateDestination(destination); err != nil {
		return Shipment{}, ErrInvalidQuoteDestination
	}
	if shippingCostCents < 0 {
		return Shipment{}, ErrInvalidTransition
	}
	now := time.Now().UTC()
	s := Shipment{
		ID:                  uuid.New().String(),
		OrderID:             orderID,
		UserID:              userID,
		MethodID:            methodID,
		ShippingCostCents:   shippingCostCents,
		DestinationSnapshot: destination,
		Status:              ShipmentStatusPending,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	s.events = []Event{ShipmentCreatedEvent{
		ShipmentID:          s.ID,
		OrderID:             s.OrderID,
		UserID:              s.UserID,
		MethodID:            s.MethodID,
		ShippingCostCents:   s.ShippingCostCents,
		DestinationSnapshot: s.DestinationSnapshot,
		CreatedAt:           s.CreatedAt,
	}}
	return s, nil
}

func (s *Shipment) Process() error {
	if s.Status != ShipmentStatusPending {
		return ErrInvalidTransition
	}
	s.Status = ShipmentStatusProcessing
	s.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Shipment) Dispatch(trackingCode, carrier string) error {
	if s.Status != ShipmentStatusProcessing {
		return ErrInvalidTransition
	}
	if trackingCode == "" || carrier == "" {
		return ErrInvalidTransition
	}
	s.Status = ShipmentStatusShipped
	s.TrackingCode = trackingCode
	s.Carrier = carrier
	now := time.Now().UTC()
	s.ShippedAt = &now
	s.UpdatedAt = now
	s.events = append(s.events, ShipmentShippedEvent{
		ShipmentID:   s.ID,
		OrderID:      s.OrderID,
		UserID:       s.UserID,
		TrackingCode: s.TrackingCode,
		Carrier:      s.Carrier,
		ShippedAt:    now,
	})
	return nil
}

func (s *Shipment) Deliver() error {
	if s.Status != ShipmentStatusShipped {
		return ErrInvalidTransition
	}
	s.Status = ShipmentStatusDelivered
	now := time.Now().UTC()
	s.DeliveredAt = &now
	s.UpdatedAt = now
	s.events = append(s.events, ShipmentDeliveredEvent{
		ShipmentID:  s.ID,
		OrderID:     s.OrderID,
		UserID:      s.UserID,
		DeliveredAt: now,
	})
	return nil
}

func (s *Shipment) Fail() error {
	if s.Status != ShipmentStatusProcessing && s.Status != ShipmentStatusShipped {
		return ErrInvalidTransition
	}
	s.Status = ShipmentStatusFailed
	s.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Shipment) CanTransitionTo(newStatus ShipmentStatus) bool {
	switch s.Status {
	case ShipmentStatusPending:
		return newStatus == ShipmentStatusProcessing
	case ShipmentStatusProcessing:
		return newStatus == ShipmentStatusShipped || newStatus == ShipmentStatusFailed
	case ShipmentStatusShipped:
		return newStatus == ShipmentStatusDelivered || newStatus == ShipmentStatusFailed
	case ShipmentStatusDelivered, ShipmentStatusFailed:
		return false
	default:
		return false
	}
}

func (s Shipment) IsTerminal() bool {
	return s.Status == ShipmentStatusDelivered || s.Status == ShipmentStatusFailed
}
