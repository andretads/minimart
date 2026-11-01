package postgres

import (
	"encoding/json"
	"fmt"
	"minimart/internal/payment/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/outbox"
)

// outboxMessage maps a domain event to the broker-agnostic outbox message. The
// key is the payment id so all of a payment's events land on one partition and
// stay ordered.
func outboxMessage(ev domain.Event) (outbox.Message, error) {
	switch e := ev.(type) {
	case domain.PaymentInitiatedEvent:
		return encode(kafka.PaymentInitiatedTopic, e.PaymentID, message.PaymentInitiated{
			PaymentID:   e.PaymentID,
			OrderID:     e.OrderID,
			UserID:      e.UserID,
			AmountCents: e.AmountCents,
			Currency:    e.Currency,
		})
	case domain.PaymentSucceededEvent:
		return encode(kafka.PaymentSucceededTopic, e.PaymentID, message.PaymentSucceeded{
			PaymentID:   e.PaymentID,
			OrderID:     e.OrderID,
			UserID:      e.UserID,
			AmountCents: e.AmountCents,
			Currency:    e.Currency,
			ProviderRef: e.ProviderRef,
		})
	case domain.PaymentFailedEvent:
		return encode(kafka.PaymentFailedTopic, e.PaymentID, message.PaymentFailed{
			PaymentID:   e.PaymentID,
			OrderID:     e.OrderID,
			UserID:      e.UserID,
			AmountCents: e.AmountCents,
			Currency:    e.Currency,
			Reason:      e.Reason,
		})
	case domain.PaymentUnknownEvent:
		return encode(kafka.PaymentUnknownTopic, e.PaymentID, message.PaymentUnknown{
			PaymentID:   e.PaymentID,
			OrderID:     e.OrderID,
			UserID:      e.UserID,
			AmountCents: e.AmountCents,
			Currency:    e.Currency,
			Reason:      e.Reason,
		})
	case domain.PaymentRefundedEvent:
		return encode(kafka.PaymentRefundedTopic, e.PaymentID, message.PaymentRefunded{
			PaymentID:   e.PaymentID,
			OrderID:     e.OrderID,
			UserID:      e.UserID,
			AmountCents: e.AmountCents,
			Currency:    e.Currency,
			ProviderRef: e.ProviderRef,
			Reason:      e.Reason,
		})
	default:
		return outbox.Message{}, fmt.Errorf("unsupported payment event %T", ev)
	}
}

func encode(topic kafka.Topic, key string, payload any) (outbox.Message, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return outbox.Message{}, fmt.Errorf("marshal %s event: %w", topic, err)
	}
	return outbox.Message{Key: key, Topic: topic.String(), Body: body}, nil
}
