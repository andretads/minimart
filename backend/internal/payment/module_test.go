package payment

import (
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	t.Run("Name returns payment", func(t *testing.T) {
		m := NewModule(nil, nil, nil, metrics.Noop{}, nil, nil, time.Hour)
		require.Equal(t, "payment", m.Name())
	})

	t.Run("implements module.Module interface", func(t *testing.T) {
		var _ module.Module = NewModule(nil, nil, nil, metrics.Noop{}, nil, nil, time.Hour)
	})
}

func TestModule_Init(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := NewModule(nil, pool, pool, metrics.Noop{}, nil, nil, time.Hour)
	require.NoError(t, m.Init(t.Context()))
	require.NotNil(t, m.svc)
	require.NotNil(t, m.handlers)
}

func TestModule_RegisterContracts(t *testing.T) {
	m := NewModule(nil, nil, nil, metrics.Noop{}, nil, nil, time.Hour)
	require.NoError(t, m.RegisterContracts(t.Context(), nil))
}

func TestModule_RegisterHTTP(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := NewModule(nil, pool, pool, metrics.Noop{}, nil, nil, time.Hour)
	require.NoError(t, m.Init(t.Context()))

	mux := http.NewServeMux()
	require.NoError(t, m.RegisterHTTP(t.Context(), mux))
}

func TestModule_RegisterWorkers(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := NewModule(nil, pool, pool, metrics.Noop{}, nil, nil, time.Hour)
	require.NoError(t, m.Init(t.Context()))

	// Just verify it doesn't panic - use a nil worker to avoid Kafka connection attempts
	require.NoError(t, m.RegisterWorkers(t.Context(), nil))
}
