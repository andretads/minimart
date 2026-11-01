//go:build integration

package test

import (
	"context"
	"encoding/json"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/internal/shipping/presentation/message/event"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShippingKafkaConsumer_ConsumesOrderPaid drives the whole input adapter
// path: a real order-paid event flows through the handler into the application
// service and lands in real Postgres, with its shipment-created event in the
// outbox.
func TestShippingKafkaConsumer_ConsumesOrderPaid(t *testing.T) {
	repo, pool := newShippingRepo(t)
	orderID := uniqueShippingID()
	cleanup(t, pool, orderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	method, err := repo.GetMethod(t.Context(), fixture.Method.ID)
	require.NoError(t, err)

	svc := application.NewService(repo, nil, nil, metrics.Noop{}, time.Hour)
	h := event.NewHandler(svc)

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	consumer := kafka.NewConsumer(testKafkaBrokers, nil, metrics.Noop{})
	errCh := make(chan error, 1)
	topic := kafka.Topic("order-paid-shipping-consumer")
	consumerGroup := "shipping-consumer-" + orderID
	go func() {
		errCh <- consumer.Subscribe(ctx, topic, consumerGroup, func(_ context.Context, msg []byte) error {
			if herr := h.OrderPaid(ctx, msg); herr != nil {
				t.Logf("handler error: %v", herr)
				return herr
			}
			cancel()
			return nil
		})
	}()

	orderPaid := message.OrderPaid{
		OrderID:          orderID,
		UserID:           "it-user",
		CustomerID:       "it-cust",
		TotalCents:       1500,
		ShippingQuoteID:  fixture.Quote.ID,
		ShippingMethodID: method.ID,
		DestinationSnapshot: message.DestinationSnapshot{
			Label: "Home", FullName: "John Doe", Street: "123 Main St", City: "Sao Paulo",
			State: "SP", Zip: "01000-000", Country: "BR", Phone: "+5511999999999",
		},
	}
	require.NoError(t, producer.PublishMessage(t.Context(), topic, orderPaid.OrderID, orderPaid))

	select {
	case <-ctx.Done():
	case subErr := <-errCh:
		require.NoError(t, subErr)
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for the order-paid message")
	}

	shipment, err := repo.GetShipmentByOrderID(t.Context(), orderID)
	require.NoError(t, err)
	assert.Equal(t, orderPaid.UserID, shipment.UserID)
	assert.Equal(t, method.ID, shipment.MethodID)
	assert.Equal(t, domain.ShipmentStatusPending, shipment.Status)

	// The shipment and its event were written together.
	records := outboxRows(t, pool, shipment.ID)
	require.Len(t, records, 1)
	assert.Equal(t, kafka.ShipmentCreatedTopic.String(), records[0].Topic)

	var created message.ShipmentCreated
	require.NoError(t, json.Unmarshal(records[0].Body, &created))
	assert.Equal(t, shipment.ID, created.ShipmentID)
	assert.Equal(t, orderID, created.OrderID)
	assert.Equal(t, int64(599), created.ShippingCostCents)
}

// TestShippingKafkaConsumer_RedeliveredOrderPaidIsIdempotent proves a
// redelivered order-paid event does not create a second shipment.
func TestShippingKafkaConsumer_RedeliveredOrderPaidIsIdempotent(t *testing.T) {
	repo, pool := newShippingRepo(t)
	orderID := uniqueShippingID()
	cleanup(t, pool, orderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	method, err := repo.GetMethod(t.Context(), fixture.Method.ID)
	require.NoError(t, err)

	svc := application.NewService(repo, nil, nil, metrics.Noop{}, time.Hour)
	h := event.NewHandler(svc)

	orderPaid := message.OrderPaid{
		OrderID:          orderID,
		UserID:           "it-user",
		CustomerID:       "it-cust",
		TotalCents:       1500,
		ShippingQuoteID:  fixture.Quote.ID,
		ShippingMethodID: method.ID,
		DestinationSnapshot: message.DestinationSnapshot{
			Label: "Home", FullName: "John Doe", Street: "123 Main St", City: "Sao Paulo",
			State: "SP", Zip: "01000-000", Country: "BR", Phone: "+5511999999999",
		},
	}
	body, err := json.Marshal(orderPaid)
	require.NoError(t, err)

	require.NoError(t, h.OrderPaid(t.Context(), body))
	require.NoError(t, h.OrderPaid(t.Context(), body))

	shipments := shipmentIDs(t, pool, orderID)
	require.Len(t, shipments, 1, "a redelivered event must not create a second shipment")

	records := outboxRows(t, pool, shipments[0])
	require.Len(t, records, 1, "a redelivered event must not record a second event")
}
