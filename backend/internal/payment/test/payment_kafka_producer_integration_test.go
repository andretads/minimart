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

func TestPaymentKafkaProducer_PublishesMessage(t *testing.T) {
	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var received []byte
	consumer := kafka.NewConsumer(testKafkaBrokers, nil, metrics.Noop{})
	errCh := make(chan error, 1)
	topic := kafka.Topic("payment-test-producer-" + uniquePaymentKey())
	go func() {
		errCh <- consumer.Subscribe(ctx, topic, "payment-producer-test-group", func(_ context.Context, msg []byte) error {
			received = msg
			cancel()
			return nil
		})
	}()

	paymentInitiated := message.PaymentInitiated{
		PaymentID:   "pay-prod-1",
		OrderID:     "order-1",
		UserID:      "user-1",
		AmountCents: 1500,
		Currency:    "USD",
	}
	require.NoError(t, producer.PublishMessage(t.Context(), topic, paymentInitiated.PaymentID, paymentInitiated))

	select {
	case <-ctx.Done():
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	require.NotNil(t, received)
	assert.Contains(t, string(received), "pay-prod-1")
}
