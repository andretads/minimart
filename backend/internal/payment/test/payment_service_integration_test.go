//go:build integration

// Package test holds the payment domain's integration tests.
package test

import (
	"minimart/internal/payment/application"
	"minimart/internal/payment/domain"
	"minimart/internal/payment/infrastructure/provider/fake"
	"minimart/pkg/metrics"
	"minimart/pkg/redis"
	"testing"
	"time"

	idemredis "minimart/pkg/idempotency/redis"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPaymentService_CreateIsIdempotent drives the whole application service over
// real Postgres and Redis: the first call charges, a replay returns the same
// payment without recharging, and the same key with different parameters is a
// conflict.
func TestPaymentService_CreateIsIdempotent(t *testing.T) {
	repo, pool := newPaymentRepo(t)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	provider := fake.New(fake.Config{DeclineAmountCents: 9999, TimeoutAmountCents: 7777})
	svc := application.NewService(repo, nil, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)

	key := uniquePaymentKey()
	req := application.CreatePaymentRequest{OrderID: "it-order", UserID: "it-user", IdempotencyKey: key, AmountCents: 1500, Currency: "USD", Method: "card"}

	first, err := svc.CreatePayment(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusSucceeded, first.Status)
	cleanup(t, pool, first.ID, key)

	// A replay returns the same payment.
	replay, err := svc.CreatePayment(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, first.ID, replay.ID)

	// The same key with a different amount is a conflict.
	req.AmountCents = 2500
	_, err = svc.CreatePayment(t.Context(), req)
	assert.ErrorIs(t, err, domain.ErrIdempotencyConflict)
}

func TestPaymentService_UnknownOnTimeout(t *testing.T) {
	repo, pool := newPaymentRepo(t)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	provider := fake.New(fake.Config{DeclineAmountCents: 9999, TimeoutAmountCents: 7777})
	svc := application.NewService(repo, nil, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)

	key := uniquePaymentKey()
	req := application.CreatePaymentRequest{OrderID: "it-order", UserID: "it-user", IdempotencyKey: key, AmountCents: 7777, Currency: "USD", Method: "card"}

	got, err := svc.CreatePayment(t.Context(), req)
	require.NoError(t, err)
	cleanup(t, pool, got.ID, key)
	assert.Equal(t, domain.StatusUnknown, got.Status, "a timeout is unknown, never failed")

	stored, err := repo.GetByID(t.Context(), got.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusUnknown, stored.Status)
}

// TestPaymentService_DeclineOnDeclineAmount tests that the fake provider declines
// the configured amount and the service records a failed payment.
func TestPaymentService_DeclineOnDeclineAmount(t *testing.T) {
	repo, pool := newPaymentRepo(t)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	provider := fake.New(fake.Config{DeclineAmountCents: 9999})
	svc := application.NewService(repo, nil, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)

	key := uniquePaymentKey()
	req := application.CreatePaymentRequest{OrderID: "it-order", UserID: "it-user", IdempotencyKey: key, AmountCents: 9999, Currency: "USD", Method: "card"}

	got, err := svc.CreatePayment(t.Context(), req)
	require.NoError(t, err)
	cleanup(t, pool, got.ID, key)
	assert.Equal(t, domain.StatusFailed, got.Status, "a decline is failed, never unknown")

	stored, err := repo.GetByID(t.Context(), got.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusFailed, stored.Status)
}

// TestPaymentService_CreateWithNoRedis tests the fail-open behavior when Redis is
// unavailable - the service should still work using only the DB unique index.
func TestPaymentService_CreateWithNoRedis(t *testing.T) {
	repo, pool := newPaymentRepo(t)

	// Use a fake Redis URL that will fail
	// This tests fail-open behavior - should use DB unique index only
	provider := fake.New(fake.Config{})
	svc := application.NewService(repo, nil, failingIdem{}, provider, metrics.Noop{}, time.Hour)

	key := uniquePaymentKey()
	req := application.CreatePaymentRequest{OrderID: "it-order", UserID: "it-user", IdempotencyKey: key, AmountCents: 1500, Currency: "USD", Method: "card"}

	first, err := svc.CreatePayment(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusSucceeded, first.Status)
	cleanup(t, pool, first.ID, key)

	// Replay should work
	replay, err := svc.CreatePayment(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, first.ID, replay.ID)

	// Different amount should conflict
	req.AmountCents = 2500
	_, err = svc.CreatePayment(t.Context(), req)
	assert.ErrorIs(t, err, domain.ErrIdempotencyConflict)
}

// TestPaymentService_RefundPayment drives the refund use case over real
// Postgres and Redis: a succeeded payment on a cancelled order is refunded,
// idempotently, and requires the order to be cancelled.
func TestPaymentService_RefundPayment(t *testing.T) {
	repo, pool := newPaymentRepo(t)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	provider := fake.New(fake.Config{})
	create := application.NewService(repo, nil, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)
	refund := application.NewService(repo, itOrderReader{status: "cancelled", totalCents: 1500}, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)
	live := application.NewService(repo, itOrderReader{status: "paid", totalCents: 1500}, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)

	key := uniquePaymentKey()
	created, err := create.CreatePayment(t.Context(), application.CreatePaymentRequest{
		OrderID: "it-order", UserID: "it-user", IdempotencyKey: key, AmountCents: 1500, Currency: "USD", Method: "card",
	})
	require.NoError(t, err)
	require.Equal(t, domain.StatusSucceeded, created.Status)
	cleanup(t, pool, created.ID, key)

	got, err := refund.RefundPayment(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusRefunded, got.Status)

	stored, err := repo.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusRefunded, stored.Status)

	// A replay of the refund is idempotent.
	again, err := refund.RefundPayment(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusRefunded, again.Status)

	// A payment whose order is not cancelled cannot be refunded.
	key2 := uniquePaymentKey()
	other, err := create.CreatePayment(t.Context(), application.CreatePaymentRequest{
		OrderID: "it-order", UserID: "it-user", IdempotencyKey: key2, AmountCents: 1500, Currency: "USD", Method: "card",
	})
	require.NoError(t, err)
	cleanup(t, pool, other.ID, key2)
	_, err = live.RefundPayment(t.Context(), other.ID)
	assert.ErrorIs(t, err, application.ErrOrderNotCancelled)
}

// TestPaymentService_RefundOrderPayments drives the order-cancelled consumer's
// use case over real Postgres: every captured payment of the order is refunded.
func TestPaymentService_RefundOrderPayments(t *testing.T) {
	repo, pool := newPaymentRepo(t)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	provider := fake.New(fake.Config{})
	create := application.NewService(repo, nil, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)
	refund := application.NewService(repo, itOrderReader{status: "cancelled", totalCents: 1500}, idemredis.New(rdb.Client), provider, metrics.Noop{}, time.Hour)

	orderID := "it-order-" + uniquePaymentKey()
	var ids []string
	for range 2 {
		key := uniquePaymentKey()
		created, err := create.CreatePayment(t.Context(), application.CreatePaymentRequest{
			OrderID: orderID, UserID: "it-user", IdempotencyKey: key, AmountCents: 1500, Currency: "USD", Method: "card",
		})
		require.NoError(t, err)
		require.Equal(t, domain.StatusSucceeded, created.Status)
		ids = append(ids, created.ID)
		cleanup(t, pool, created.ID, key)
	}

	require.NoError(t, refund.RefundOrderPayments(t.Context(), orderID))
	for _, id := range ids {
		stored, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusRefunded, stored.Status)
	}

	// At-least-once delivery: replaying the event refunds nothing new and errors nothing.
	require.NoError(t, refund.RefundOrderPayments(t.Context(), orderID))
}
