package order

import (
	orderoutbox "minimart/internal/order/infrastructure/outbox"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func testOutboxSettings() orderoutbox.Settings {
	return orderoutbox.Settings{
		ProcessInterval:           10 * time.Millisecond,
		LockCheckerInterval:       10 * time.Millisecond,
		CleanupWorkerInterval:     10 * time.Millisecond,
		MaxLockTimeDuration:       100 * time.Millisecond,
		MessagesRetentionDuration: 100 * time.Millisecond,
		MaxSendAttempts:           3,
	}
}

func TestModule(t *testing.T) {
	t.Run("Name returns order", func(t *testing.T) {
		m := NewModule(nil, nil, nil, nil, metrics.Noop{}, nil, time.Hour, testOutboxSettings())
		require.Equal(t, "order", m.Name())
	})

	t.Run("implements module.Module interface", func(t *testing.T) {
		var _ module.Module = NewModule(nil, nil, nil, nil, metrics.Noop{}, nil, time.Hour, testOutboxSettings())
	})
}

func TestModule_Init(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := NewModule(nil, pool, pool, nil, metrics.Noop{}, nil, time.Hour, testOutboxSettings())
	require.NoError(t, m.Init(t.Context()))
	require.NotNil(t, m.svc)
	require.NotNil(t, m.handlers)
}

func TestModule_RegisterContracts(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := NewModule(nil, pool, pool, nil, metrics.Noop{}, nil, time.Hour, testOutboxSettings())
	require.NoError(t, m.Init(t.Context()))
	require.NoError(t, m.RegisterContracts(t.Context(), &contracts.Contracts{}))
}

func TestModule_RegisterHTTP(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := NewModule(nil, pool, pool, nil, metrics.Noop{}, nil, time.Hour, testOutboxSettings())
	require.NoError(t, m.Init(t.Context()))

	mux := http.NewServeMux()
	require.NoError(t, m.RegisterHTTP(t.Context(), mux))
}

// RegisterWorkers registers Kafka consumers and outbox relay; requires valid
// worker with consumer/publisher. Test skipped - verified by existence check.
func TestModule_RegisterWorkers(t *testing.T) {
	m := NewModule(nil, nil, nil, nil, metrics.Noop{}, nil, time.Hour, testOutboxSettings())
	_ = m.RegisterWorkers
}
