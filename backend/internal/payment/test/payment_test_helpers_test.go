//go:build integration

// Package test holds the payment domain's integration tests.
package test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	orderclient "minimart/internal/order/presentation/module/client"
	"minimart/internal/payment/application"
	"minimart/internal/payment/domain"
	"minimart/internal/payment/infrastructure/postgres"
	"minimart/internal/payment/infrastructure/provider/fake"
	"minimart/pkg/idempotency"
	"minimart/pkg/kafka"
	"minimart/pkg/metrics"
	"minimart/pkg/redis"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	idemredis "minimart/pkg/idempotency/redis"
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

// mutableOrderReader answers a fixed order whose status may flip after a number
// of calls, simulating an order cancelled while a charge is in flight.
type mutableOrderReader struct {
	order      orderclient.Order
	err        error
	calls      int
	flipAfter  int
	flipStatus string
}

func (r *mutableOrderReader) GetOrder(context.Context, string) (orderclient.Order, error) {
	r.calls++
	if r.err != nil {
		return orderclient.Order{}, r.err
	}
	o := r.order
	if r.flipAfter > 0 && r.calls > r.flipAfter {
		o.Status = r.flipStatus
	}
	return o, nil
}

// failingSaveRepo injects a storage failure before the real repository writes.
type failingSaveRepo struct {
	domain.Repository
	err error
}

func (r failingSaveRepo) Save(context.Context, domain.Payment) error { return r.err }

// failingUpdateRepo injects a failure when a charge is being finalized.
type failingUpdateRepo struct {
	domain.Repository
	err error
}

func (r failingUpdateRepo) Update(context.Context, domain.Payment) error { return r.err }

// blockingProvider stops the first authorize call so a concurrent request can
// reach the database while the charge is still in flight.
type blockingProvider struct {
	domain.Provider
	once    sync.Once
	entered chan struct{}
	release chan struct{}
}

func newBlockingProvider(p domain.Provider) *blockingProvider {
	return &blockingProvider{
		Provider: p,
		entered:  make(chan struct{}),
		release:  make(chan struct{}),
	}
}

func (p *blockingProvider) Authorize(ctx context.Context, req domain.AuthorizationRequest) (domain.Result, error) {
	p.once.Do(func() { close(p.entered) })
	select {
	case <-p.release:
	case <-ctx.Done():
		return domain.Result{}, ctx.Err()
	}
	return p.Provider.Authorize(ctx, req)
}

// failingRefundProvider delegates to a provider but fails every refund.
type failingRefundProvider struct {
	domain.Provider
	err error
}

func (p failingRefundProvider) Refund(context.Context, string, int64) (domain.Result, error) {
	return domain.Result{}, p.err
}

// paymentEnv wires the application service over the real Postgres repository and
// Redis idempotency store, with a fake provider and a scriptable order reader.
type paymentEnv struct {
	svc      *application.Service
	repo     domain.Repository
	pgRepo   *postgres.PaymentRepository
	pool     *pgxpool.Pool
	reader   *mutableOrderReader
	provider domain.Provider
	idem     idempotency.Store
	orderID  string
	userID   string
}

func newPaymentEnv(t *testing.T) *paymentEnv {
	t.Helper()

	pgRepo, pool := newPaymentRepo(t)

	rdb, err := redis.New(testRedisURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	orderID := "o1-" + uniquePaymentKey()
	userID := "u1-" + uniquePaymentKey()
	e := &paymentEnv{
		repo:   pgRepo,
		pgRepo: pgRepo,
		pool:   pool,
		reader: &mutableOrderReader{order: orderclient.Order{ID: orderID, UserID: userID, Status: "placed", TotalCents: 1500, Currency: "USD"}},
		provider: fake.New(fake.Config{
			DeclineAmountCents:           9999,
			TimeoutAmountCents:           7777,
			SettleThenTimeoutAmountCents: 8888,
		}),
		idem:    idemredis.New(rdb.Client),
		orderID: orderID,
		userID:  userID,
	}
	e.wire()
	return e
}

func (e *paymentEnv) wire() {
	var reader application.OrderReader
	if e.reader != nil {
		reader = e.reader
	}
	e.svc = application.NewService(e.repo, reader, e.idem, e.provider, metrics.Noop{}, time.Hour)
}

func (e *paymentEnv) request() application.CreatePaymentRequest {
	return application.CreatePaymentRequest{
		OrderID:        e.orderID,
		UserID:         e.userID,
		IdempotencyKey: uniquePaymentKey(),
		AmountCents:    1500,
		Currency:       "USD",
		Method:         "card",
	}
}

// createPayment runs the use case and schedules cleanup of the created row.
func (e *paymentEnv) createPayment(t *testing.T, req application.CreatePaymentRequest) domain.Payment {
	t.Helper()
	p, err := e.svc.CreatePayment(t.Context(), req)
	require.NoError(t, err)
	cleanup(t, e.pool, p.ID, req.IdempotencyKey)
	return p
}

func (e *paymentEnv) stored(t *testing.T, id string) domain.Payment {
	t.Helper()
	p, err := e.pgRepo.GetByID(t.Context(), id)
	require.NoError(t, err)
	return p
}

func (e *paymentEnv) reconcile(t *testing.T) {
	t.Helper()
	time.Sleep(2 * time.Millisecond)
	reconciler := application.NewReconciler(e.svc, time.Millisecond, 10, slog.Default())
	reconciler.RunOnce(t.Context())
}

func (e *paymentEnv) setOrderStatus(status string) {
	e.reader.order.Status = status
}

// outboxRow is one persisted payment event.
type outboxRow struct {
	Topic string
	Body  []byte
}

func outboxRows(t *testing.T, pool *pgxpool.Pool, paymentID string) []outboxRow {
	t.Helper()

	rows, err := pool.Query(t.Context(), "SELECT topic, body FROM outbox WHERE message_key = $1", paymentID)
	require.NoError(t, err)
	defer rows.Close()

	var out []outboxRow
	for rows.Next() {
		var row outboxRow
		require.NoError(t, rows.Scan(&row.Topic, &row.Body))
		out = append(out, row)
	}
	require.NoError(t, rows.Err())
	return out
}

func countTopic(rows []outboxRow, topic kafka.Topic) int {
	count := 0
	for _, row := range rows {
		if row.Topic == topic.String() {
			count++
		}
	}
	return count
}
