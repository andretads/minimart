package showcase

import (
	"minimart/pkg/module"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	t.Run("Name returns showcase", func(t *testing.T) {
		m := NewModule(nil)
		require.Equal(t, "showcase", m.Name())
	})

	t.Run("implements module.Module interface", func(t *testing.T) {
		var _ module.Module = NewModule(nil)
	})

	t.Run("Init builds service and handlers", func(t *testing.T) {
		// Use a test pool that will fail on actual queries but allows Init to succeed
		pool, err := pgxpool.New(t.Context(), "postgres://invalid")
		require.NoError(t, err) // pool creation succeeds, connection fails lazily
		defer pool.Close()

		m := NewModule(pool)
		require.NoError(t, m.Init(t.Context()))
		require.NotNil(t, m.svc)
		require.NotNil(t, m.handlers)
	})

	t.Run("RegisterContracts is no-op", func(t *testing.T) {
		m := NewModule(nil)
		require.NoError(t, m.RegisterContracts(t.Context(), nil))
	})

	t.Run("RegisterHTTP registers routes without error", func(t *testing.T) {
		pool, err := pgxpool.New(t.Context(), "postgres://invalid")
		require.NoError(t, err)
		defer pool.Close()

		m := NewModule(pool)
		require.NoError(t, m.Init(t.Context()))

		mux := http.NewServeMux()
		require.NoError(t, m.RegisterHTTP(t.Context(), mux))
		// If RegisterHTTP succeeds, routes are registered
	})

	t.Run("RegisterWorkers is no-op", func(t *testing.T) {
		m := NewModule(nil)
		require.NoError(t, m.RegisterWorkers(t.Context(), nil))
	})
}
