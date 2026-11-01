//go:build integration

package test

import (
	"context"
	"encoding/json"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"testing"
	"time"

	orderoutbox "minimart/internal/order/infrastructure/outbox"
	outboxpg "minimart/pkg/outbox/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOutboxRelayPublishesPendingRecord seeds a pending outbox row and proves the
// relay publishes it to Kafka and marks it delivered - the end-to-end path from
// the shared outbox table to the broker, against real Postgres and Kafka.
func TestOutboxRelayPublishesPendingRecord(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	key := uniqueOrderID()
	body, err := json.Marshal(message.OrderPlaced{
		OrderID:    key,
		UserID:     "it-user",
		TotalCents: 500,
		ItemCount:  1,
	})
	require.NoError(t, err)

	_, err = pool.Exec(t.Context(),
		`INSERT INTO outbox (id, topic, message_key, body, state, created_on)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		"it-outbox-"+key, kafka.OrderPlacedTopic.String(), key, body, 0, time.Now().UTC())
	require.NoError(t, err)
	t.Cleanup(func() {
		//nolint:usetesting // t.Context() is cancelled before cleanup runs
		_, _ = pool.Exec(context.Background(), "DELETE FROM outbox WHERE message_key = $1", key)
	})

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })
	consumer := kafka.NewConsumer(testKafkaBrokers, producer, metrics.Noop{})

	received := make(chan []byte, 1)
	subCtx, subCancel := context.WithCancel(t.Context())
	defer subCancel()
	go func() {
		_ = consumer.Subscribe(subCtx, kafka.OrderPlacedTopic, "it-outbox-relay-"+key, func(_ context.Context, msg []byte) error {
			select {
			case received <- msg:
			default:
			}
			return nil
		})
	}()

	relay := orderoutbox.NewRelay(outboxpg.NewStore(pool), producer, "it-relay", orderoutbox.Settings{
		ProcessInterval:           50 * time.Millisecond,
		LockCheckerInterval:       time.Hour,
		CleanupWorkerInterval:     time.Hour,
		MaxLockTimeDuration:       time.Minute,
		MessagesRetentionDuration: time.Hour,
		MaxSendAttempts:           3,
	})
	relayCtx, relayCancel := context.WithCancel(t.Context())
	defer relayCancel()
	go relay.Run(relayCtx)

	// Wait for message with a reasonable timeout (5s should be plenty with 50ms polling)
	select {
	case msg := <-received:
		var got message.OrderPlaced
		require.NoError(t, json.Unmarshal(msg, &got))
		assert.Equal(t, key, got.OrderID)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the relayed order-placed event")
	}

	// After receiving the message, the relay should have marked the record delivered.
	// Poll briefly to account for the async DB update.
	require.Eventually(t, func() bool {
		var state int
		if err := pool.QueryRow(t.Context(), "SELECT state FROM outbox WHERE message_key = $1", key).Scan(&state); err != nil {
			return false
		}
		return state == 1
	}, 2*time.Second, 50*time.Millisecond, "record should be marked delivered")
}
