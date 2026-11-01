package domain

import "time"

// Event is a domain event raised by the Shipping domain. Infrastructure maps
// each concrete event to a serialized broker payload in the same transaction
// that persists the shipment, so events are published at least once via the
// outbox.
type Event interface {
	// isShippingEvent seals the interface to this package's event types.
	isShippingEvent()
}

// ShipmentCreatedEvent records that a shipment was created for a paid order.
type ShipmentCreatedEvent struct {
	ShipmentID          string
	OrderID             string
	UserID              string
	MethodID            string
	ShippingCostCents   int64
	DestinationSnapshot DestinationSnapshot
	CreatedAt           time.Time
}

func (ShipmentCreatedEvent) isShippingEvent() {}

// ShipmentShippedEvent records that a shipment was dispatched.
type ShipmentShippedEvent struct {
	ShipmentID   string
	OrderID      string
	UserID       string
	TrackingCode string
	Carrier      string
	ShippedAt    time.Time
}

func (ShipmentShippedEvent) isShippingEvent() {}

// ShipmentDeliveredEvent records that a shipment was delivered.
type ShipmentDeliveredEvent struct {
	ShipmentID  string
	OrderID     string
	UserID      string
	DeliveredAt time.Time
}

func (ShipmentDeliveredEvent) isShippingEvent() {}
