//go:build integration

// Package test holds the payment domain's integration tests.
package test

import (
	"context"
	"encoding/json"
	"minimart/internal/payment/application"
	"minimart/internal/payment/domain"
	"minimart/internal/payment/infrastructure/provider/fake"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"minimart/pkg/redis"
	"testing"
	"time"

	idemredis "minimart/pkg/idempotency/redis"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPaymentOutboxRelayPublishesPaymentSucceeded creates a payment (which writes its events to
// the outbox) and proves the relay publishes the PaymentSucceeded event to Kafka.
func TestPaymentOutboxRelayPublishesPaymentSucceeded(t *testing.T) {
	repo, pool := newPaymentRepo(t)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	provider := fake.New(fake.Config{})
	svc := application.NewService(repo, nil, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)

	key := uniquePaymentKey()
	created, err := svc.CreatePayment(t.Context(), application.CreatePaymentRequest{
		OrderID: "it-order", UserID: "it-user", IdempotencyKey: key, AmountCents: 1500, Currency: "USD", Method: "card",
	})
	require.NoError(t, err)
	require.Equal(t, domain.StatusSucceeded, created.Status)
	cleanup(t, pool, created.ID, key)

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })
	consumer := kafka.NewConsumer(testKafkaBrokers, producer, metrics.Noop{})

	received := make(chan []byte, 1)
	subCtx, subCancel := context.WithCancel(t.Context())
	defer subCancel()
	go func() {
		_ = consumer.Subscribe(subCtx, kafka.PaymentSucceededTopic, "it-pay-relay-"+key, func(ctx context.Context, msg []byte) error {
			select {
			case received <- msg:
			case <-ctx.Done():
			}
			return nil
		})
	}()

	relay := newPaymentRelay(t, pool, producer)
	startPaymentRelay(t, relay)

	select {
	case msg := <-received:
		var got message.PaymentSucceeded
		require.NoError(t, json.Unmarshal(msg, &got))
		assert.Equal(t, created.ID, got.PaymentID)
		assert.Equal(t, int64(1500), got.AmountCents)
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for the relayed payment-succeeded event")
	}

	require.Eventually(t, func() bool {
		var state int
		if err := pool.QueryRow(t.Context(), "SELECT state FROM outbox WHERE message_key = $1 AND topic = $2", created.ID, kafka.PaymentSucceededTopic.String()).Scan(&state); err != nil {
			return false
		}
		return state == 1
	}, 10*time.Second, 100*time.Millisecond, "record should be marked delivered")
}

// TestPaymentOutboxRelayPublishesPaymentFailed proves the relay publishes PaymentFailed events.
func TestPaymentOutboxRelayPublishesPaymentFailed(t *testing.T) {
	repo, pool := newPaymentRepo(t)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	provider := fake.New(fake.Config{DeclineAmountCents: 9999})
	svc := application.NewService(repo, nil, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)

	key := uniquePaymentKey()
	created, err := svc.CreatePayment(t.Context(), application.CreatePaymentRequest{
		OrderID: "it-order", UserID: "it-user", IdempotencyKey: key, AmountCents: 9999, Currency: "USD", Method: "card",
	})
	require.NoError(t, err)
	require.Equal(t, domain.StatusFailed, created.Status)
	cleanup(t, pool, created.ID, key)

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })
	consumer := kafka.NewConsumer(testKafkaBrokers, producer, metrics.Noop{})

	received := make(chan []byte, 1)
	subCtx, subCancel := context.WithCancel(t.Context())
	defer subCancel()
	go func() {
		_ = consumer.Subscribe(subCtx, kafka.PaymentFailedTopic, "it-pay-relay-failed-"+key, func(ctx context.Context, msg []byte) error {
			select {
			case received <- msg:
			case <-ctx.Done():
			}
			return nil
		})
	}()

	relay := newPaymentRelay(t, pool, producer)
	startPaymentRelay(t, relay)

	select {
	case msg := <-received:
		var got message.PaymentFailed
		require.NoError(t, json.Unmarshal(msg, &got))
		assert.Equal(t, created.ID, got.PaymentID)
		assert.Equal(t, int64(9999), got.AmountCents)
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for the relayed payment-failed event")
	}

	require.Eventually(t, func() bool {
		var state int
		if err := pool.QueryRow(t.Context(), "SELECT state FROM outbox WHERE message_key = $1 AND topic = $2", created.ID, kafka.PaymentFailedTopic.String()).Scan(&state); err != nil {
			return false
		}
		return state == 1
	}, 10*time.Second, 100*time.Millisecond, "record should be marked delivered")
}

// TestPaymentOutboxRelayPublishesPaymentUnknown proves the relay publishes PaymentUnknown events.
func TestPaymentOutboxRelayPublishesPaymentUnknown(t *testing.T) {
	repo, pool := newPaymentRepo(t)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	provider := fake.New(fake.Config{TimeoutAmountCents: 7777})
	svc := application.NewService(repo, nil, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)

	key := uniquePaymentKey()
	created, err := svc.CreatePayment(t.Context(), application.CreatePaymentRequest{
		OrderID: "it-order", UserID: "it-user", IdempotencyKey: key, AmountCents: 7777, Currency: "USD", Method: "card",
	})
	require.NoError(t, err)
	require.Equal(t, domain.StatusUnknown, created.Status)
	cleanup(t, pool, created.ID, key)

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })
	consumer := kafka.NewConsumer(testKafkaBrokers, producer, metrics.Noop{})

	received := make(chan []byte, 1)
	subCtx, subCancel := context.WithCancel(t.Context())
	defer subCancel()
	go func() {
		_ = consumer.Subscribe(subCtx, kafka.PaymentUnknownTopic, "it-pay-relay-unknown-"+key, func(ctx context.Context, msg []byte) error {
			select {
			case received <- msg:
			case <-ctx.Done():
			}
			return nil
		})
	}()

	relay := newPaymentRelay(t, pool, producer)
	startPaymentRelay(t, relay)

	select {
	case msg := <-received:
		var got message.PaymentUnknown
		require.NoError(t, json.Unmarshal(msg, &got))
		assert.Equal(t, created.ID, got.PaymentID)
		assert.Equal(t, int64(7777), got.AmountCents)
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for the relayed payment-unknown event")
	}

	require.Eventually(t, func() bool {
		var state int
		if err := pool.QueryRow(t.Context(), "SELECT state FROM outbox WHERE message_key = $1 AND topic = $2", created.ID, kafka.PaymentUnknownTopic.String()).Scan(&state); err != nil {
			return false
		}
		return state == 1
	}, 10*time.Second, 100*time.Millisecond, "record should be marked delivered")
}

// TestPaymentOutboxRelayPublishesPaymentRefunded proves the relay publishes the
// PaymentRefunded event written by a refund.
func TestPaymentOutboxRelayPublishesPaymentRefunded(t *testing.T) {
	repo, pool := newPaymentRepo(t)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	provider := fake.New(fake.Config{})
	create := application.NewService(repo, nil, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)
	refund := application.NewService(repo, itOrderReader{status: "cancelled", totalCents: 1500}, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)

	key := uniquePaymentKey()
	created, err := create.CreatePayment(t.Context(), application.CreatePaymentRequest{
		OrderID: "it-order", UserID: "it-user", IdempotencyKey: key, AmountCents: 1500, Currency: "USD", Method: "card",
	})
	require.NoError(t, err)
	require.Equal(t, domain.StatusSucceeded, created.Status)
	refunded, err := refund.RefundPayment(t.Context(), created.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusRefunded, refunded.Status)
	cleanup(t, pool, created.ID, key)

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })
	consumer := kafka.NewConsumer(testKafkaBrokers, producer, metrics.Noop{})

	received := make(chan []byte, 1)
	subCtx, subCancel := context.WithCancel(t.Context())
	defer subCancel()
	go func() {
		_ = consumer.Subscribe(subCtx, kafka.PaymentRefundedTopic, "it-pay-relay-refunded-"+key, func(ctx context.Context, msg []byte) error {
			select {
			case received <- msg:
			case <-ctx.Done():
			}
			return nil
		})
	}()

	relay := newPaymentRelay(t, pool, producer)
	startPaymentRelay(t, relay)

	select {
	case msg := <-received:
		var got message.PaymentRefunded
		require.NoError(t, json.Unmarshal(msg, &got))
		if got.PaymentID != created.ID {
			t.Fatalf("unexpected payment id in relayed event: %s", got.PaymentID)
		}
		assert.Equal(t, "it-order", got.OrderID)
		assert.Equal(t, int64(1500), got.AmountCents)
		assert.Equal(t, "reembolso solicitado", got.Reason)
		require.NotEmpty(t, got.ProviderRef)
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for the relayed payment-refunded event")
	}

	require.Eventually(t, func() bool {
		var state int
		if err := pool.QueryRow(t.Context(), "SELECT state FROM outbox WHERE message_key = $1 AND topic = $2", created.ID, kafka.PaymentRefundedTopic.String()).Scan(&state); err != nil {
			return false
		}
		return state == 1
	}, 10*time.Second, 100*time.Millisecond, "record should be marked delivered")
}

// TestPaymentOutboxRelayPublishesPaymentInitiated proves the relay publishes PaymentInitiated events.
func TestPaymentOutboxRelayPublishesPaymentInitiated(t *testing.T) {
	repo, pool := newPaymentRepo(t)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	provider := fake.New(fake.Config{})
	svc := application.NewService(repo, nil, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)

	key := uniquePaymentKey()
	created, err := svc.CreatePayment(t.Context(), application.CreatePaymentRequest{
		OrderID: "it-order", UserID: "it-user", IdempotencyKey: key, AmountCents: 1500, Currency: "USD", Method: "card",
	})
	require.NoError(t, err)
	require.Equal(t, domain.StatusSucceeded, created.Status)
	cleanup(t, pool, created.ID, key)

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })
	consumer := kafka.NewConsumer(testKafkaBrokers, producer, metrics.Noop{})

	received := make(chan []byte, 1)
	subCtx, subCancel := context.WithCancel(t.Context())
	defer subCancel()
	go func() {
		_ = consumer.Subscribe(subCtx, kafka.PaymentInitiatedTopic, "it-pay-relay-initiated-"+key, func(ctx context.Context, msg []byte) error {
			select {
			case received <- msg:
			case <-ctx.Done():
			}
			return nil
		})
	}()

	relay := newPaymentRelay(t, pool, producer)
	startPaymentRelay(t, relay)

	deadline := time.After(60 * time.Second)
	for {
		select {
		case msg := <-received:
			var got message.PaymentInitiated
			require.NoError(t, json.Unmarshal(msg, &got))
			if got.PaymentID != created.ID {
				continue
			}
			assert.Equal(t, int64(1500), got.AmountCents)
			goto delivered
		case <-deadline:
			t.Fatal("timed out waiting for the relayed payment-initiated event after 60s")
		}
	}

delivered:

	require.Eventually(t, func() bool {
		var state int
		if err := pool.QueryRow(t.Context(), "SELECT state FROM outbox WHERE message_key = $1 AND topic = $2", created.ID, kafka.PaymentInitiatedTopic.String()).Scan(&state); err != nil {
			return false
		}
		return state == 1
	}, 10*time.Second, 100*time.Millisecond, "record should be marked delivered")
}
