//go:build integration

package test

import (
	"context"
	"encoding/json"
	"minimart/internal/shipping/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// awaitShipmentCreated waits for the shipment-created event of the given
// shipment. The relay publishes every pending outbox row, so other shipments'
// events may arrive first.
func awaitShipmentCreated(t *testing.T, received <-chan []byte, shipmentID string) message.ShipmentCreated {
	t.Helper()
	deadline := time.After(60 * time.Second)
	for {
		select {
		case msg := <-received:
			var got message.ShipmentCreated
			require.NoError(t, json.Unmarshal(msg, &got))
			if got.ShipmentID != shipmentID {
				continue
			}
			return got
		case <-deadline:
			t.Fatal("timed out waiting for the relayed shipment-created event")
			return message.ShipmentCreated{}
		}
	}
}

// awaitShipmentShipped waits for the shipment-shipped event of the given
// shipment.
func awaitShipmentShipped(t *testing.T, received <-chan []byte, shipmentID string) message.ShipmentShipped {
	t.Helper()
	deadline := time.After(60 * time.Second)
	for {
		select {
		case msg := <-received:
			var got message.ShipmentShipped
			require.NoError(t, json.Unmarshal(msg, &got))
			if got.ShipmentID != shipmentID {
				continue
			}
			return got
		case <-deadline:
			t.Fatal("timed out waiting for the relayed shipment-shipped event")
			return message.ShipmentShipped{}
		}
	}
}

// awaitShipmentDelivered waits for the shipment-delivered event of the given
// shipment.
func awaitShipmentDelivered(t *testing.T, received <-chan []byte, shipmentID string) message.ShipmentDelivered {
	t.Helper()
	deadline := time.After(60 * time.Second)
	for {
		select {
		case msg := <-received:
			var got message.ShipmentDelivered
			require.NoError(t, json.Unmarshal(msg, &got))
			if got.ShipmentID != shipmentID {
				continue
			}
			return got
		case <-deadline:
			t.Fatal("timed out waiting for the relayed shipment-delivered event")
			return message.ShipmentDelivered{}
		}
	}
}

// TestShippingOutboxRelayPublishesShipmentCreated creates a shipment (which
// writes its event to the outbox) and proves the relay publishes the
// shipment-created event to Kafka.
func TestShippingOutboxRelayPublishesShipmentCreated(t *testing.T) {
	repo, pool := newShippingRepo(t)
	orderID := uniqueShippingID()
	cleanup(t, pool, orderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	shipment, err := domain.NewShipment(orderID, "it-user", fixture.Method.ID, fixture.Quote.DestinationSnapshot, 599)
	require.NoError(t, err)
	require.NoError(t, repo.CreateShipment(t.Context(), shipment))

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })
	consumer := kafka.NewConsumer(testKafkaBrokers, producer, metrics.Noop{})

	received := make(chan []byte, 16)
	subCtx, subCancel := context.WithCancel(t.Context())
	defer subCancel()
	go func() {
		_ = consumer.Subscribe(subCtx, kafka.ShipmentCreatedTopic, "it-ship-relay-"+orderID, func(_ context.Context, msg []byte) error {
			select {
			case received <- msg:
			default:
			}
			return nil
		})
	}()

	relay := newShippingRelay(t, pool, producer)
	relayCtx, relayCancel := context.WithCancel(t.Context())
	defer relayCancel()
	go relay.Run(relayCtx)

	got := awaitShipmentCreated(t, received, shipment.ID)
	assert.Equal(t, shipment.ID, got.ShipmentID)
	assert.Equal(t, orderID, got.OrderID)
	assert.Equal(t, int64(599), got.ShippingCostCents)

	require.Eventually(t, func() bool {
		var state int
		if err := pool.QueryRow(t.Context(), "SELECT state FROM outbox WHERE message_key = $1 AND topic = $2",
			shipment.ID, kafka.ShipmentCreatedTopic.String()).Scan(&state); err != nil {
			return false
		}
		return state == 1
	}, 10*time.Second, 100*time.Millisecond, "record should be marked delivered")
}

// TestShippingOutboxRelayPublishesShipmentShippedAndDelivered proves the
// shipment lifecycle events reach Kafka too.
func TestShippingOutboxRelayPublishesShipmentShippedAndDelivered(t *testing.T) {
	repo, pool := newShippingRepo(t)
	orderID := uniqueShippingID()
	cleanup(t, pool, orderID)

	fixture := seedShippingFixture(t, repo, "it-cust")
	cleanup(t, pool, fixture.ids()...)

	shipment, err := domain.NewShipment(orderID, "it-user", fixture.Method.ID, fixture.Quote.DestinationSnapshot, 599)
	require.NoError(t, err)
	require.NoError(t, repo.CreateShipment(t.Context(), shipment))

	// The use cases load, transition and persist the shipment again.
	processing, err := repo.GetShipment(t.Context(), shipment.ID)
	require.NoError(t, err)
	require.NoError(t, processing.Process())
	require.NoError(t, processing.Dispatch("TRACK123", "Correios"))
	require.NoError(t, repo.UpdateShipment(t.Context(), processing))

	shipped, err := repo.GetShipment(t.Context(), shipment.ID)
	require.NoError(t, err)
	require.NoError(t, shipped.Deliver())
	require.NoError(t, repo.UpdateShipment(t.Context(), shipped))

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })
	consumer := kafka.NewConsumer(testKafkaBrokers, producer, metrics.Noop{})

	shippedMsgs := make(chan []byte, 16)
	deliveredMsgs := make(chan []byte, 16)
	subCtx, subCancel := context.WithCancel(t.Context())
	defer subCancel()
	go func() {
		_ = consumer.Subscribe(subCtx, kafka.ShipmentShippedTopic, "it-ship-relay-shipped-"+orderID, func(_ context.Context, msg []byte) error {
			select {
			case shippedMsgs <- msg:
			default:
			}
			return nil
		})
	}()
	go func() {
		_ = consumer.Subscribe(subCtx, kafka.ShipmentDeliveredTopic, "it-ship-relay-delivered-"+orderID, func(_ context.Context, msg []byte) error {
			select {
			case deliveredMsgs <- msg:
			default:
			}
			return nil
		})
	}()

	relay := newShippingRelay(t, pool, producer)
	relayCtx, relayCancel := context.WithCancel(t.Context())
	defer relayCancel()
	go relay.Run(relayCtx)

	gotShipped := awaitShipmentShipped(t, shippedMsgs, shipment.ID)
	assert.Equal(t, shipment.ID, gotShipped.ShipmentID)
	assert.Equal(t, "TRACK123", gotShipped.TrackingCode)
	assert.Equal(t, "Correios", gotShipped.Carrier)

	gotDelivered := awaitShipmentDelivered(t, deliveredMsgs, shipment.ID)
	assert.Equal(t, shipment.ID, gotDelivered.ShipmentID)
	assert.Equal(t, orderID, gotDelivered.OrderID)
}
