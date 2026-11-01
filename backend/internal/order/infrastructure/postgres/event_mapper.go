package postgres

import (
	"encoding/json"
	"fmt"
	"minimart/internal/order/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/outbox"
)

// outboxMessage maps a domain event to the broker-agnostic outbox message. The
// key is the order id so all of an order's events land on one partition and stay
// ordered.
func outboxMessage(ev domain.Event) (outbox.Message, error) {
	switch e := ev.(type) {
	case domain.OrderPlacedEvent:
		return encode(kafka.OrderPlacedTopic, e.OrderID, message.OrderPlaced{
			OrderID:    e.OrderID,
			UserID:     e.UserID,
			CustomerID: e.CustomerID,
			TotalCents: e.TotalCents,
			ItemCount:  e.ItemCount,
		})
	case domain.OrderCancelledEvent:
		return encode(kafka.OrderCancelledTopic, e.OrderID, message.OrderCancelled{
			OrderID:    e.OrderID,
			UserID:     e.UserID,
			CustomerID: e.CustomerID,
			TotalCents: e.TotalCents,
			ItemCount:  e.ItemCount,
		})
	case domain.OrderStockReleaseEvent:
		return encode(kafka.OrderStockReleaseTopic, e.OrderID, message.OrderStockRelease{
			OrderID: e.OrderID,
		})
	case domain.OrderPaidEvent:
		return encode(kafka.OrderPaidTopic, e.OrderID, message.OrderPaid{
			OrderID:             e.OrderID,
			UserID:              e.UserID,
			CustomerID:          e.CustomerID,
			TotalCents:          e.TotalCents,
			ShippingQuoteID:     e.ShippingQuoteID,
			ShippingMethodID:    e.ShippingMethodID,
			DestinationSnapshot: e.DestinationSnapshot,
		})
	default:
		return outbox.Message{}, fmt.Errorf("unsupported order event %T", ev)
	}
}

func encode(topic kafka.Topic, key string, payload any) (outbox.Message, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return outbox.Message{}, fmt.Errorf("marshal %s event: %w", topic, err)
	}
	return outbox.Message{Key: key, Topic: topic.String(), Body: body}, nil
}
