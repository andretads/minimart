//go:build integration

package test

import (
	"context"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationProducer_PublishesMessage(t *testing.T) {
	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var received []byte
	consumer := kafka.NewConsumer(testKafkaBrokers, nil, metrics.Noop{})
	errCh := make(chan error, 1)
	topic := kafka.Topic("order-placed-notification-producer")
	go func() {
		errCh <- consumer.Subscribe(ctx, topic, "notification-producer-test-group", func(_ context.Context, msg []byte) error {
			received = msg
			cancel()
			return nil
		})
	}()

	orderPlaced := message.OrderPlaced{
		OrderID:    "prod-test-1",
		UserID:     "user-1",
		TotalCents: 2500,
		ItemCount:  3,
	}
	require.NoError(t, producer.PublishMessage(t.Context(), topic, orderPlaced.OrderID, orderPlaced))

	select {
	case <-ctx.Done():
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	require.NotNil(t, received)
	assert.Contains(t, string(received), "prod-test-1")
}
