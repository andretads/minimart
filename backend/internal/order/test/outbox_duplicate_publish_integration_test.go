//go:build integration

package test

import (
	"context"
	"minimart/pkg/outbox"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	outboxpg "minimart/pkg/outbox/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockingBroker struct {
	sends    atomic.Int32
	release  chan struct{}
	released sync.Once
}

func newBlockingBroker() *blockingBroker {
	return &blockingBroker{release: make(chan struct{})}
}

func (b *blockingBroker) Send(ctx context.Context, _ outbox.Message) error {
	b.sends.Add(1)
	select {
	case <-b.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (b *blockingBroker) unblock() {
	b.released.Do(func() { close(b.release) })
}

// TestOutboxDispatchersSharingAMachineIDPublishOnce proves that two dispatchers
// built from the same host identity - one per domain inside the worker process -
// publish a pending record exactly once instead of once per dispatcher.
func TestOutboxDispatchersSharingAMachineIDPublishOnce(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	key := uniqueOrderID()
	_, err = pool.Exec(t.Context(),
		`INSERT INTO outbox (id, topic, message_key, body, state, created_on)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		"it-outbox-dup-"+key, "order-placed", key, []byte(`{}`), 0, time.Now().UTC())
	require.NoError(t, err)
	t.Cleanup(func() {
		//nolint:usetesting // t.Context() is cancelled before cleanup runs
		_, _ = pool.Exec(context.Background(), "DELETE FROM outbox WHERE message_key = $1", key)
	})

	broker := newBlockingBroker()
	store := outboxpg.NewStore(pool)
	settings := outbox.DispatcherSettings{
		ProcessInterval:           10 * time.Millisecond,
		LockCheckerInterval:       time.Hour,
		CleanupWorkerInterval:     time.Hour,
		MaxLockTimeDuration:       time.Minute,
		MessagesRetentionDuration: time.Hour,
	}

	first := outbox.NewDispatcher(store, broker, settings, "it-shared-host")
	second := outbox.NewDispatcher(store, broker, settings, "it-shared-host")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go first.Run(ctx)
	go second.Run(ctx)

	require.Eventually(t, func() bool { return broker.sends.Load() >= 1 }, 5*time.Second, 10*time.Millisecond)
	broker.unblock()

	require.Never(t, func() bool { return broker.sends.Load() > 1 }, time.Second, 20*time.Millisecond,
		"a record must be published once, not once per dispatcher")
	assert.Equal(t, int32(1), broker.sends.Load())
}
