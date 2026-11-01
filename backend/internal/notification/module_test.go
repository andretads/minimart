package notification

import (
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	t.Run("Name returns notification", func(t *testing.T) {
		m := NewModule(nil, &contracts.Contracts{}, nil, nil, nil, nil, metrics.Noop{})
		require.Equal(t, "notification", m.Name())
	})

	t.Run("implements module.Module interface", func(t *testing.T) {
		var _ module.Module = NewModule(nil, &contracts.Contracts{}, nil, nil, nil, nil, metrics.Noop{})
	})
}

func TestModule_Init(t *testing.T) {
	m := NewModule(nil, &contracts.Contracts{}, nil, nil, nil, nil, metrics.Noop{})

	_ = m.Init
}

func TestModule_RegisterContracts(t *testing.T) {
	m := NewModule(nil, &contracts.Contracts{}, nil, nil, nil, nil, metrics.Noop{})
	require.NoError(t, m.RegisterContracts(t.Context(), nil))
}

func TestModule_RegisterHTTP(t *testing.T) {
	m := NewModule(nil, &contracts.Contracts{}, nil, nil, nil, nil, metrics.Noop{})

	mux := http.NewServeMux()
	require.NoError(t, m.RegisterHTTP(t.Context(), mux))
}

func TestModule_RegisterWorkers(t *testing.T) {
	m := NewModule(nil, &contracts.Contracts{}, nil, nil, nil, nil, metrics.Noop{})
	_ = m.RegisterWorkers
}
