//go:build integration

package test

import (
	"context"
	"encoding/json"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogConsumer_ConsumesMessage(t *testing.T) {
	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var received sync.Map
	consumer := kafka.NewConsumer(testKafkaBrokers, nil, metrics.Noop{})
	errCh := make(chan error, 1)
	topic := kafka.Topic("order-placed-catalog-consumer")
	go func() {
		errCh <- consumer.Subscribe(ctx, topic, "catalog-consumer-test-group", func(_ context.Context, msg []byte) error {
			received.Store("msg", msg)
			cancel()
			return nil
		})
	}()

	orderPlaced := message.OrderPlaced{
		OrderID:    "cat-cons-1",
		UserID:     "cat-user-2",
		TotalCents: 2000,
		ItemCount:  4,
	}
	require.NoError(t, producer.PublishMessage(t.Context(), topic, orderPlaced.OrderID, orderPlaced))

	select {
	case <-ctx.Done():
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	v, ok := received.Load("msg")
	require.True(t, ok)
	body, ok := v.([]byte)
	require.True(t, ok)
	var got message.OrderPlaced
	require.NoError(t, json.Unmarshal(body, &got))
	assert.Equal(t, orderPlaced, got)
}
