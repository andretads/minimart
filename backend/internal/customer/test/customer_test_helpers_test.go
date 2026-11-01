//go:build integration

package test

import (
	"context"
	"crypto/rand"
	"minimart/internal/customer/application"
	"minimart/internal/customer/domain"
	"minimart/internal/customer/infrastructure/postgres"
	"minimart/pkg/metrics"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func newCustomerServices(t *testing.T) *application.Service {
	t.Helper()
	readPool, writePool := mustPools(t)
	repo := postgres.NewCustomerRepository(readPool, writePool)
	return application.NewService(repo, metrics.Noop{})
}

func newTestUserID() string {
	return "test-user-" + rand.Text()
}

func validCreateCustomerRequest() application.CreateCustomerRequest {
	return application.CreateCustomerRequest{
		UserID:      "user-1",
		DisplayName: "John",
	}
}

func createTestCustomer(t *testing.T, svc *application.Service, req application.CreateCustomerRequest) domain.Customer {
	t.Helper()
	c, err := svc.CreateCustomer(t.Context(), req)
	require.NoError(t, err)
	return c
}

func getTestCustomer(t *testing.T, svc *application.Service, id string) domain.Customer {
	t.Helper()
	c, err := svc.GetCustomerByID(t.Context(), id)
	require.NoError(t, err)
	return c
}

func mustPools(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	ctx := t.Context()
	readPool, err := newPool(ctx, testDSN)
	require.NoError(t, err)
	writePool, err := newPool(ctx, testDSN)
	require.NoError(t, err)
	return readPool, writePool
}

func newPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}
