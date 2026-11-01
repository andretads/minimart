package postgres

import (
	"encoding/json"
	"fmt"
	"minimart/internal/shipping/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/outbox"
	"time"
)

// outboxMessage maps a domain event to the broker-agnostic outbox message. The
// key is the shipment id so all of a shipment's events land on one partition
// and stay ordered.
func outboxMessage(ev domain.Event) (outbox.Message, error) {
	switch e := ev.(type) {
	case domain.ShipmentCreatedEvent:
		return encode(kafka.ShipmentCreatedTopic, e.ShipmentID, message.ShipmentCreated{
			ShipmentID:        e.ShipmentID,
			OrderID:           e.OrderID,
			UserID:            e.UserID,
			MethodID:          e.MethodID,
			ShippingCostCents: e.ShippingCostCents,
			DestinationSnapshot: message.DestinationSnapshot{
				Label:    e.DestinationSnapshot.Label,
				FullName: e.DestinationSnapshot.FullName,
				Street:   e.DestinationSnapshot.Street,
				City:     e.DestinationSnapshot.City,
				State:    e.DestinationSnapshot.State,
				Zip:      e.DestinationSnapshot.Zip,
				Country:  e.DestinationSnapshot.Country,
				Phone:    e.DestinationSnapshot.Phone,
			},
			CreatedAt: e.CreatedAt.Format(time.RFC3339),
		})
	case domain.ShipmentShippedEvent:
		return encode(kafka.ShipmentShippedTopic, e.ShipmentID, message.ShipmentShipped{
			ShipmentID:   e.ShipmentID,
			OrderID:      e.OrderID,
			UserID:       e.UserID,
			TrackingCode: e.TrackingCode,
			Carrier:      e.Carrier,
			ShippedAt:    e.ShippedAt.Format(time.RFC3339),
		})
	case domain.ShipmentDeliveredEvent:
		return encode(kafka.ShipmentDeliveredTopic, e.ShipmentID, message.ShipmentDelivered{
			ShipmentID:  e.ShipmentID,
			OrderID:     e.OrderID,
			UserID:      e.UserID,
			DeliveredAt: e.DeliveredAt.Format(time.RFC3339),
		})
	default:
		return outbox.Message{}, fmt.Errorf("unsupported shipping event %T", ev)
	}
}

func encode(topic kafka.Topic, key string, payload any) (outbox.Message, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return outbox.Message{}, fmt.Errorf("marshal %s event: %w", topic, err)
	}
	return outbox.Message{Key: key, Topic: topic.String(), Body: body}, nil
}
