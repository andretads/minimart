//go:build integration

// Package test holds the payment domain's integration tests.
package test

import (
	"context"
	"errors"
	"fmt"
	orderclient "minimart/internal/order/presentation/module/client"
	"minimart/internal/payment/infrastructure/postgres"
	"minimart/pkg/idempotency"
	"minimart/pkg/kafka"
	"sync/atomic"
	"testing"
	"time"

	paymentoutbox "minimart/internal/payment/infrastructure/outbox"
	outboxpkg "minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// uniquePaymentSeq disambiguates keys generated within the same clock tick.
var uniquePaymentSeq atomic.Int64

// uniquePaymentKey returns a value unlikely to collide with a previous run's
// rows, used for both the payment id and its idempotency key.
func uniquePaymentKey() string {
	return fmt.Sprintf("it-pay-%d-%d", time.Now().UnixNano(), uniquePaymentSeq.Add(1))
}

// newPaymentRepo connects to the real database and returns the adapter. The read
// and write pools point at the same database, and the repository writes domain
// events to the shared outbox via the real Publisher.
func newPaymentRepo(t *testing.T) (*postgres.PaymentRepository, *pgxpool.Pool) {
	t.Helper()

	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	store := outboxpg.NewStore(pool)
	return postgres.NewPaymentRepository(pool, pool, outboxpkg.NewPublisher(store)), pool
}

// cleanup deletes the payment and any outbox events keyed to it after the test.
// t.Context() is cancelled before cleanup runs, so use a fresh context.
func cleanup(t *testing.T, pool *pgxpool.Pool, id, key string) {
	t.Helper()
	t.Cleanup(func() {
		//nolint:usetesting // t.Context() is cancelled before cleanup runs
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "DELETE FROM payments WHERE idempotency_key = $1", key)
		_, _ = pool.Exec(ctx, "DELETE FROM outbox WHERE message_key = $1", id)
	})
}

// newPaymentRelay builds the payment outbox relay over the real shared outbox
// store with fast intervals suited to a test.
func newPaymentRelay(t *testing.T, pool *pgxpool.Pool, publisher kafka.Publisher) *paymentoutbox.Relay {
	t.Helper()
	return paymentoutbox.NewRelay(outboxpg.NewStore(pool), publisher, "it-relay", paymentoutbox.Settings{
		ProcessInterval:           50 * time.Millisecond,
		LockCheckerInterval:       time.Hour,
		CleanupWorkerInterval:     time.Hour,
		MaxLockTimeDuration:       time.Minute,
		MessagesRetentionDuration: time.Hour,
		MaxSendAttempts:           3,
	})
}

// startPaymentRelay runs the relay in the background and registers a cleanup
// that cancels it and waits for it to stop. Waiting is essential: a relay that
// outlives its test can lock outbox rows belonging to the next test, which then
// never get published.
func startPaymentRelay(t *testing.T, relay *paymentoutbox.Relay) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		relay.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

// failingIdem simulates an unavailable idempotency store: every operation
// returns an error so the service's fail-open path falls back to the database.
type failingIdem struct{}

func (failingIdem) Get(context.Context, string) (idempotency.Record, bool, error) {
	return idempotency.Record{}, false, errors.New("idempotency store unavailable")
}

func (failingIdem) Reserve(context.Context, string, string, time.Duration) (bool, error) {
	return false, errors.New("idempotency store unavailable")
}

func (failingIdem) Complete(context.Context, string, string, string, time.Duration) error {
	return errors.New("idempotency store unavailable")
}

func (failingIdem) Release(context.Context, string, string) error {
	return errors.New("idempotency store unavailable")
}

// itOrderReader answers a fixed order for the payment service's pre-charge
// validation and refund checks.
type itOrderReader struct {
	status     string
	totalCents int64
}

func (r itOrderReader) GetOrder(context.Context, string) (orderclient.Order, error) {
	return orderclient.Order{
		ID:         "it-order",
		UserID:     "it-user",
		Status:     r.status,
		TotalCents: r.totalCents,
		Currency:   "USD",
	}, nil
}
