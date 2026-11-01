//go:build integration

// Package test holds the order domain's integration tests.
package test

import (
	"context"
	"fmt"
	"minimart/internal/order/infrastructure/postgres"
	"minimart/pkg/idempotency"
	"sync"
	"testing"
	"time"

	outboxpkg "minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// uniqueOrderID returns an id unlikely to collide with a previous run's rows.
// orders.id is a plain primary key (no upsert), so a fixed id would fail on the
// second run.
func uniqueOrderID() string {
	return fmt.Sprintf("it-order-%d", time.Now().UnixNano())
}

// newOrderRepo connects to the real database and returns the adapter. The read
// and write pools point at the same database, and the repository writes domain
// events to the shared outbox via the real Publisher.
func newOrderRepo(t *testing.T) (*postgres.OrderRepository, *pgxpool.Pool) {
	t.Helper()

	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	store := outboxpg.NewStore(pool)
	return postgres.NewOrderRepository(pool, pool, outboxpkg.NewPublisher(store)), pool
}

// cleanup deletes the order, its items, and any outbox events keyed to it after
// the test. t.Context() is cancelled before cleanup runs, so use a fresh context.
func cleanup(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()
	t.Cleanup(func() {
		//nolint:usetesting // t.Context() is cancelled before cleanup runs
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "DELETE FROM order_items WHERE order_id = $1", id)
		_, _ = pool.Exec(ctx, "DELETE FROM orders WHERE id = $1", id)
		_, _ = pool.Exec(ctx, "DELETE FROM outbox WHERE message_key = $1", id)
	})
}

// outboxTopics returns the topics of the outbox rows keyed to the given order id.
func outboxTopics(ctx context.Context, t *testing.T, pool *pgxpool.Pool, key string) []string {
	t.Helper()
	rows, err := pool.Query(ctx, "SELECT topic FROM outbox WHERE message_key = $1", key)
	require.NoError(t, err)
	defer rows.Close()
	var topics []string
	for rows.Next() {
		var topic string
		require.NoError(t, rows.Scan(&topic))
		topics = append(topics, topic)
	}
	require.NoError(t, rows.Err())
	return topics
}

// countOutbox returns how many outbox rows are keyed to the given order id.
func countOutbox(ctx context.Context, t *testing.T, pool *pgxpool.Pool, key string) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM outbox WHERE message_key = $1", key).Scan(&n))
	return n
}

// outboxBody returns the payload of the outbox row written for the given order
// id and topic.
func outboxBody(ctx context.Context, t *testing.T, pool *pgxpool.Pool, key string, topic string) []byte {
	t.Helper()
	var body []byte
	err := pool.QueryRow(ctx, "SELECT body FROM outbox WHERE message_key = $1 AND topic = $2", key, topic).Scan(&body)
	require.NoError(t, err)
	return body
}

// memIdem is an in-memory idempotency.Store. Service tests use this instead of
// the Redis adapter; the Redis namespace regression test starts its own container.
type memIdem struct {
	mu   sync.Mutex
	recs map[string]idempotency.Record
}

func newMemIdem() *memIdem {
	return &memIdem{recs: make(map[string]idempotency.Record)}
}

func (m *memIdem) Get(_ context.Context, key string) (idempotency.Record, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.recs[key]
	return rec, ok, nil
}

func (m *memIdem) Reserve(_ context.Context, key, fingerprint string, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.recs[key]; ok {
		return false, nil
	}
	m.recs[key] = idempotency.Record{Fingerprint: fingerprint}
	return true, nil
}

func (m *memIdem) Complete(_ context.Context, key, fingerprint, resourceID string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recs[key] = idempotency.Record{Fingerprint: fingerprint, ResourceID: resourceID}
	return nil
}

func (m *memIdem) Release(_ context.Context, key, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.recs, key)
	return nil
}
