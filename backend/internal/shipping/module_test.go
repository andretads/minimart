package shipping

import (
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/infrastructure/memory"
	shippingoutbox "minimart/internal/shipping/infrastructure/outbox"
	shippinghttp "minimart/internal/shipping/presentation/http"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModule_Init(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil, metrics.Noop{}, time.Hour)
	m := NewModule(nil, nil, nil, nil, metrics.Noop{}, nil, time.Hour, shippingoutbox.Settings{})
	m.svc = svc
	m.handlers = shippinghttp.NewHandlers(svc, nil)

	assert.Implements(t, (*module.Module)(nil), m)
	assert.Equal(t, "shipping", m.Name())
}

func TestModule_RegisterHTTP(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil, metrics.Noop{}, time.Hour)
	m := NewModule(nil, nil, nil, nil, metrics.Noop{}, nil, time.Hour, shippingoutbox.Settings{})
	m.svc = svc
	m.handlers = shippinghttp.NewHandlers(svc, nil)

	mux := http.NewServeMux()
	err := m.RegisterHTTP(t.Context(), mux)
	require.NoError(t, err)
}

func TestModule_RegisterWorkers(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil, metrics.Noop{}, time.Hour)
	m := NewModule(nil, nil, nil, nil, metrics.Noop{}, nil, time.Hour, shippingoutbox.Settings{})
	m.svc = svc
	m.handlers = shippinghttp.NewHandlers(svc, nil)

	// Just verify it doesn't panic - use a nil worker to avoid Kafka connection attempts
	err := m.RegisterWorkers(t.Context(), nil)
	require.NoError(t, err)
}
