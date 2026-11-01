package postgres

import (
	"context"
	"errors"
	"fmt"
	"minimart/pkg/outbox"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	relayLockNamespace  = "minimart:outbox:relay"
	relayLockKey        = int32(0)
	relayLockCloseLimit = 5 * time.Second
)

// AdvisoryRelayLock serializes outbox publish cycles across worker replicas.
type AdvisoryRelayLock struct {
	pool *pgxpool.Pool

	mu   sync.Mutex
	conn *pgxpool.Conn
}

var _ outbox.RelayLock = (*AdvisoryRelayLock)(nil)

// NewAdvisoryRelayLock builds a Postgres advisory-lock backed outbox relay lock.
func NewAdvisoryRelayLock(pool *pgxpool.Pool) *AdvisoryRelayLock {
	return &AdvisoryRelayLock{pool: pool}
}

func (l *AdvisoryRelayLock) TryAcquire(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conn != nil {
		return true, nil
	}

	conn, err := l.pool.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("acquire outbox relay connection: %w", err)
	}

	acquired, err := New(conn).TryAcquireRelayLock(ctx, TryAcquireRelayLockParams{
		Namespace: relayLockNamespace,
		Key:       relayLockKey,
	})
	if err != nil {
		conn.Release()
		return false, fmt.Errorf("try acquire outbox advisory lock: %w", err)
	}
	if !acquired {
		conn.Release()
		return false, nil
	}

	l.conn = conn
	return true, nil
}

func (l *AdvisoryRelayLock) Release(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conn == nil {
		return nil
	}

	conn := l.conn
	l.conn = nil

	released, err := New(conn).ReleaseRelayLock(ctx, ReleaseRelayLockParams{
		Namespace: relayLockNamespace,
		Key:       relayLockKey,
	})
	if err != nil || !released {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), relayLockCloseLimit)
		_ = conn.Conn().Close(closeCtx)
		cancel()
	}
	conn.Release()

	if err != nil {
		return fmt.Errorf("release outbox advisory lock: %w", err)
	}
	if !released {
		return errors.New("release outbox advisory lock: lock was not held")
	}
	return nil
}
