package kafka

import (
	"context"
	"minimart/pkg/outbox"
)

// OutboxBroker adapts the Kafka Publisher to the outbox.MessageBroker port, so an
// outbox dispatcher can relay stored messages without importing sarama/watermill.
type OutboxBroker struct {
	publisher Publisher
}

var _ outbox.MessageBroker = (*OutboxBroker)(nil)

// NewOutboxBroker wraps publisher as a message broker for the outbox dispatcher.
func NewOutboxBroker(publisher Publisher) *OutboxBroker {
	return &OutboxBroker{publisher: publisher}
}

// Send publishes a stored outbox message. PublishMessage already treats []byte
// payloads as the raw body, so the stored bytes go out unchanged.
func (b *OutboxBroker) Send(ctx context.Context, msg outbox.Message) error {
	return b.publisher.PublishMessage(ctx, Topic(msg.Topic), msg.Key, msg.Body)
}
