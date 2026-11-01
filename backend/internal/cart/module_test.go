package cart

import (
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	t.Run("Name returns cart", func(t *testing.T) {
		m := NewModule(nil, nil, metrics.Noop{})
		require.Equal(t, "cart", m.Name())
	})

	t.Run("implements module.Module interface", func(t *testing.T) {
		var _ module.Module = NewModule(nil, nil, metrics.Noop{})
	})
}

func TestModule_Init(t *testing.T) {
	m := NewModule(nil, nil, metrics.Noop{})
	// Init requires real MongoDB; just verify method exists and doesn't panic on construction
	_ = m.Init
}

func TestModule_RegisterContracts(t *testing.T) {
	m := NewModule(nil, nil, metrics.Noop{})
	require.NoError(t, m.RegisterContracts(t.Context(), nil))
}

func TestModule_RegisterHTTP(t *testing.T) {
	m := NewModule(nil, nil, metrics.Noop{})
	// RegisterHTTP without Init verifies method exists
	mux := http.NewServeMux()
	require.NoError(t, m.RegisterHTTP(t.Context(), mux))
}

// RegisterWorkers is no-op (server-only module)
func TestModule_RegisterWorkers(t *testing.T) {
	m := NewModule(nil, nil, metrics.Noop{})
	require.NoError(t, m.RegisterWorkers(t.Context(), nil))
}
