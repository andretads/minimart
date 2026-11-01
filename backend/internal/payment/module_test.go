package payment

import (
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"net/http"
	"testing"
	"time"

	paymentoutbox "minimart/internal/payment/infrastructure/outbox"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func testOutboxSettings() paymentoutbox.Settings {
	return paymentoutbox.Settings{
		ProcessInterval:           10 * time.Millisecond,
		LockCheckerInterval:       10 * time.Millisecond,
		CleanupWorkerInterval:     10 * time.Millisecond,
		MaxLockTimeDuration:       100 * time.Millisecond,
		MessagesRetentionDuration: 100 * time.Millisecond,
		MaxSendAttempts:           3,
	}
}

func TestModule(t *testing.T) {
	t.Run("Name returns payment", func(t *testing.T) {
		m := NewModule(nil, nil, nil, nil, metrics.Noop{}, nil, nil, time.Hour, testOutboxSettings())
		require.Equal(t, "payment", m.Name())
	})

	t.Run("implements module.Module interface", func(t *testing.T) {
		var _ module.Module = NewModule(nil, nil, nil, nil, metrics.Noop{}, nil, nil, time.Hour, testOutboxSettings())
	})
}

func TestModule_Init(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := NewModule(nil, pool, pool, nil, metrics.Noop{}, nil, nil, time.Hour, testOutboxSettings())
	require.NoError(t, m.Init(t.Context()))
	require.NotNil(t, m.svc)
	require.NotNil(t, m.handlers)
}

func TestModule_RegisterContracts(t *testing.T) {
	m := NewModule(nil, nil, nil, nil, metrics.Noop{}, nil, nil, time.Hour, testOutboxSettings())
	require.NoError(t, m.RegisterContracts(t.Context(), nil))
}

func TestModule_RegisterHTTP(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := NewModule(nil, pool, pool, nil, metrics.Noop{}, nil, nil, time.Hour, testOutboxSettings())
	require.NoError(t, m.Init(t.Context()))

	mux := http.NewServeMux()
	require.NoError(t, m.RegisterHTTP(t.Context(), mux))
}

func TestModule_RegisterWorkers(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := NewModule(nil, pool, pool, nil, metrics.Noop{}, nil, nil, time.Hour, testOutboxSettings())
	require.NoError(t, m.Init(t.Context()))

	// Just verify it doesn't panic - use a nil worker to avoid Kafka connection attempts
	require.NoError(t, m.RegisterWorkers(t.Context(), nil))
}
