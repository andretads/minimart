//go:build integration

package test

import (
	"context"
	"crypto/rand"
	"minimart/internal/address/domain"
	"minimart/pkg/metrics"
	"testing"

	addressapp "minimart/internal/address/application"
	addresspg "minimart/internal/address/infrastructure/postgres"
	customerapp "minimart/internal/customer/application"
	customerpg "minimart/internal/customer/infrastructure/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func newAddressServices(t *testing.T) (*addressapp.Service, *customerapp.Service) {
	t.Helper()
	readPool, writePool := mustPools(t)
	addressRepo := addresspg.NewAddressRepository(readPool, writePool)
	customerRepo := customerpg.NewCustomerRepository(readPool, writePool)
	return addressapp.NewService(addressRepo, metrics.Noop{}), customerapp.NewService(customerRepo, metrics.Noop{})
}

func newTestCustomer(t *testing.T, svc *customerapp.Service) string {
	t.Helper()
	c, err := svc.CreateCustomer(t.Context(), customerapp.CreateCustomerRequest{
		UserID:      "test-user-" + rand.Text(),
		DisplayName: "Test User",
	})
	require.NoError(t, err)
	return c.ID
}

func validCreateAddressRequest(customerID string) addressapp.CreateAddressRequest {
	return addressapp.CreateAddressRequest{
		CustomerID: customerID,
		Label:      "Home",
		FullName:   "John Doe",
		Street:     "123 Main St",
		City:       "New York",
		State:      "NY",
		Zip:        "10001",
		Country:    "USA",
		Phone:      "+1-555-1234",
		IsDefault:  false,
	}
}

func validUpdateAddressRequest(id, customerID string, version int) addressapp.UpdateAddressRequest {
	return addressapp.UpdateAddressRequest{
		ID:         id,
		CustomerID: customerID,
		Label:      "Home",
		FullName:   "John Doe",
		Street:     "123 Main St",
		City:       "New York",
		State:      "NY",
		Zip:        "10001",
		Country:    "USA",
		Phone:      "+1-555-1234",
		Version:    version,
	}
}

func createTestAddress(t *testing.T, svc *addressapp.Service, req addressapp.CreateAddressRequest) domain.Address {
	t.Helper()
	a, err := svc.CreateAddress(t.Context(), req)
	require.NoError(t, err)
	return a
}

func getTestAddress(t *testing.T, svc *addressapp.Service, id, customerID string) domain.Address {
	t.Helper()
	a, err := svc.GetAddress(t.Context(), id, customerID)
	require.NoError(t, err)
	return a
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
