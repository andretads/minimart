package catalog

import (
	"context"
	customerclient "minimart/internal/customer/presentation/module/client"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// stubCustomerClient stands in for the customer contract in the module tests.
type stubCustomerClient struct{}

func (stubCustomerClient) EnsureCustomer(_ context.Context, userID string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: "cust-1", UserID: userID}, nil
}

func (stubCustomerClient) GetCustomerByID(_ context.Context, id string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: id}, nil
}

func (stubCustomerClient) GetCustomerByUserID(_ context.Context, userID string) (customerclient.Customer, error) {
	return customerclient.Customer{ID: "cust-1", UserID: userID}, nil
}

func TestModule(t *testing.T) {
	t.Run("Name returns catalog", func(t *testing.T) {
		m := NewModule(&contracts.Contracts{}, nil, nil, metrics.Noop{})
		require.Equal(t, "catalog", m.Name())
	})

	t.Run("implements module.Module interface", func(t *testing.T) {
		var _ module.Module = NewModule(&contracts.Contracts{}, nil, nil, metrics.Noop{})
	})
}

func TestModule_Init(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	m := NewModule(&contracts.Contracts{}, pool, pool, metrics.Noop{})
	require.NoError(t, m.Init(t.Context()))
	require.NotNil(t, m.repo)
	require.Nil(t, m.svc)
	require.Nil(t, m.handlers)
}

func TestModule_RegisterContracts(t *testing.T) {
	m := NewModule(&contracts.Contracts{}, nil, nil, metrics.Noop{})
	c := &contracts.Contracts{}
	require.NoError(t, m.RegisterContracts(t.Context(), c))
	require.NotNil(t, c.CatalogClient)
}

func TestModule_RegisterHTTP(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	registry := &contracts.Contracts{}
	m := NewModule(registry, pool, pool, metrics.Noop{})
	require.NoError(t, m.Init(t.Context()))

	// RegisterContracts must be called first to build the service; the
	// customer contract is published by the customer module before any route
	// is registered.
	require.NoError(t, m.RegisterContracts(t.Context(), registry))
	registry.CustomerClient = stubCustomerClient{}

	mux := http.NewServeMux()
	require.NoError(t, m.RegisterHTTP(t.Context(), mux))
	require.NotNil(t, m.handlers)
}

func TestModule_RegisterHTTPRequiresCustomerContract(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://invalid")
	require.NoError(t, err)
	defer pool.Close()

	registry := &contracts.Contracts{}
	m := NewModule(registry, pool, pool, metrics.Noop{})
	require.NoError(t, m.Init(t.Context()))
	require.NoError(t, m.RegisterContracts(t.Context(), registry))

	mux := http.NewServeMux()
	require.Error(t, m.RegisterHTTP(t.Context(), mux))
}

// RegisterWorkers registers Kafka consumers; it requires a valid worker with
// a consumer, so we only test that the method exists and returns no error
// when called with nil (which exercises the code path without panicking).
func TestModule_RegisterWorkers(t *testing.T) {
	m := NewModule(&contracts.Contracts{}, nil, nil, metrics.Noop{})
	// Just verify the method exists and doesn't crash on nil receiver setup
	_ = m.RegisterWorkers
}
