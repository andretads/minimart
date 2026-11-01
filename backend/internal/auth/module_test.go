package auth

import (
	"minimart/pkg/auth"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func newTestModule(t *testing.T, readPool, writePool *pgxpool.Pool) *Module {
	t.Helper()
	tokens, _ := auth.NewTokenService("test-secret", 15*time.Minute, 7*24*time.Hour)
	encryption, _ := auth.NewCookieEncryption([]byte("0123456789abcdef0123456789abcdef"))
	cookieConfig := auth.DefaultCookieConfig(15*time.Minute, 7*24*time.Hour)
	return NewModule(nil, readPool, writePool, tokens, encryption, cookieConfig, 30*24*time.Hour, metrics.Noop{})
}

func TestModule(t *testing.T) {
	t.Run("Name returns auth", func(t *testing.T) {
		m := newTestModule(t, nil, nil)
		require.Equal(t, "auth", m.Name())
	})

	t.Run("implements module.Module interface", func(t *testing.T) {
		var _ module.Module = newTestModule(t, nil, nil)
	})
}

func TestModule_Init(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := newTestModule(t, pool, pool)
	require.NoError(t, m.Init(t.Context()))
	require.NotNil(t, m.svc)
	require.NotNil(t, m.handlers)
}

func TestModule_RegisterContracts(t *testing.T) {
	m := newTestModule(t, nil, nil)
	require.NoError(t, m.RegisterContracts(t.Context(), &contracts.Contracts{}))
}

func TestModule_RegisterHTTP(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := newTestModule(t, pool, pool)
	require.NoError(t, m.Init(t.Context()))

	mux := http.NewServeMux()
	require.NoError(t, m.RegisterHTTP(t.Context(), mux))
}

func TestModule_RegisterWorkers(t *testing.T) {
	m := newTestModule(t, nil, nil)
	require.NoError(t, m.RegisterWorkers(t.Context(), nil))
}
