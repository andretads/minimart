package customer

import (
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	t.Run("Name returns customer", func(t *testing.T) {
		m := NewModule(nil, nil, metrics.Noop{})
		require.Equal(t, "customer", m.Name())
	})

	t.Run("implements module.Module interface", func(t *testing.T) {
		var _ module.Module = NewModule(nil, nil, metrics.Noop{})
	})
}

func TestModule_Init(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := NewModule(pool, pool, metrics.Noop{})
	require.NoError(t, m.Init(t.Context()))
	require.NotNil(t, m.svc)
	require.NotNil(t, m.handlers)
}

func TestModule_RegisterContracts(t *testing.T) {
	m := NewModule(nil, nil, metrics.Noop{})
	c := &contracts.Contracts{}
	require.NoError(t, m.RegisterContracts(t.Context(), c))
}

func TestModule_RegisterHTTP(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := NewModule(pool, pool, metrics.Noop{})
	require.NoError(t, m.Init(t.Context()))

	mux := http.NewServeMux()
	require.NoError(t, m.RegisterHTTP(t.Context(), mux))
}

func TestModule_RegisterWorkers(t *testing.T) {
	m := NewModule(nil, nil, metrics.Noop{})
	require.NoError(t, m.RegisterWorkers(t.Context(), nil))
}
