//go:build integration

package test

import (
	"context"
	addressapp "minimart/internal/address/application"
	"minimart/internal/address/domain"
	addresspg "minimart/internal/address/infrastructure/postgres"
	customerapp "minimart/internal/customer/application"
	customerpg "minimart/internal/customer/infrastructure/postgres"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddressIntegration(t *testing.T) {
	readPool, writePool := mustPools(t)
	addressRepo := addresspg.NewAddressRepository(readPool, writePool)
	customerRepo := customerpg.NewCustomerRepository(readPool, writePool)
	addressSvc := addressapp.NewService(addressRepo, metrics.Noop{})
	customerSvc := customerapp.NewService(customerRepo, metrics.Noop{})

	// Create a test customer to use for all address tests
	testCustomer, err := customerSvc.CreateCustomer(t.Context(), customerapp.CreateCustomerRequest{
		UserID:      "test-user-" + uniqueID(),
		DisplayName: "Test User",
	})
	require.NoError(t, err)
	testCustomerID := testCustomer.ID

	t.Run("create and get address by ID", func(t *testing.T) {
		created, err := addressSvc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{
			CustomerID: testCustomerID,
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			Phone:      "+1-555-1234",
			IsDefault:  false,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, created.ID)
		assert.Equal(t, "Home", created.Label)
		assert.Equal(t, "John Doe", created.FullName)
		assert.Equal(t, 1, created.Version)

		got, err := addressSvc.GetAddress(t.Context(), created.ID, testCustomerID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Home", got.Label)
		assert.Equal(t, "John Doe", got.FullName)

		// Clean up to avoid polluting other tests
		err = addressSvc.DeleteAddress(t.Context(), addressapp.DeleteAddressRequest{
			ID:         created.ID,
			CustomerID: testCustomerID,
		})
		require.NoError(t, err)
	})

	t.Run("list addresses ordered by is_default DESC, created_at DESC", func(t *testing.T) {
		_, err := addressSvc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{
			CustomerID: testCustomerID,
			Label:      "Work",
			FullName:   "John Doe",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			IsDefault:  false,
		})
		require.NoError(t, err)

		_, err = addressSvc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{
			CustomerID: testCustomerID,
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			IsDefault:  true,
		})
		require.NoError(t, err)

		_, err = addressSvc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{
			CustomerID: testCustomerID,
			Label:      "Other",
			FullName:   "John Doe",
			Street:     "789 Other St",
			City:       "Chicago",
			State:      "IL",
			Zip:        "60601",
			Country:    "USA",
			IsDefault:  false,
		})
		require.NoError(t, err)

		got, err := addressSvc.ListAddresses(t.Context(), addressapp.ListAddressesRequest{
			CustomerID: testCustomerID,
		})
		require.NoError(t, err)
		require.Len(t, got, 3)
		// Default should be first
		assert.Equal(t, "Home", got[0].Label)
		assert.True(t, got[0].IsDefault)
		// Then the rest in created_at DESC order
		assert.Equal(t, "Other", got[1].Label)
		assert.Equal(t, "Work", got[2].Label)
	})

	t.Run("get non-existent returns ErrAddressNotFound", func(t *testing.T) {
		_, err := addressSvc.GetAddress(t.Context(), "non-existent", testCustomerID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("update address", func(t *testing.T) {
		created, err := addressSvc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{
			CustomerID: testCustomerID,
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.NoError(t, err)

		updated, err := addressSvc.UpdateAddress(t.Context(), addressapp.UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: testCustomerID,
			Label:      "Work",
			FullName:   "Jane Smith",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Phone:      "+1-555-5678",
			Version:    created.Version,
		})
		require.NoError(t, err)
		assert.Equal(t, "Work", updated.Label)
		assert.Equal(t, "Jane Smith", updated.FullName)
		assert.Equal(t, 2, updated.Version)
		assert.WithinDuration(t, created.CreatedAt, updated.CreatedAt, 1*time.Second)

		got, err := addressSvc.GetAddress(t.Context(), created.ID, testCustomerID)
		require.NoError(t, err)
		assert.Equal(t, "Work", got.Label)
		assert.Equal(t, "Jane Smith", got.FullName)
	})

	t.Run("delete address", func(t *testing.T) {
		created, err := addressSvc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{
			CustomerID: testCustomerID,
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.NoError(t, err)

		err = addressSvc.DeleteAddress(t.Context(), addressapp.DeleteAddressRequest{
			ID:         created.ID,
			CustomerID: testCustomerID,
		})
		require.NoError(t, err)

		_, err = addressSvc.GetAddress(t.Context(), created.ID, testCustomerID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("set default address unsets old default", func(t *testing.T) {
		// Create a fresh customer for this test to avoid conflicts with other tests
		freshCustomer, err := customerSvc.CreateCustomer(t.Context(), customerapp.CreateCustomerRequest{
			UserID:      "test-user-" + uniqueID(),
			DisplayName: "Fresh User",
		})
		require.NoError(t, err)
		freshCustomerID := freshCustomer.ID

		// Create first address as default
		first, err := addressSvc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{
			CustomerID: freshCustomerID,
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
			IsDefault:  true,
		})
		require.NoError(t, err)
		assert.True(t, first.IsDefault)

		// Create second address
		second, err := addressSvc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{
			CustomerID: freshCustomerID,
			Label:      "Work",
			FullName:   "John Doe",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			IsDefault:  false,
		})
		require.NoError(t, err)
		assert.False(t, second.IsDefault)

		// Set second as default
		updated, err := addressSvc.SetDefaultAddress(t.Context(), addressapp.SetDefaultAddressRequest{
			ID:         second.ID,
			CustomerID: freshCustomerID,
		})
		require.NoError(t, err)
		assert.True(t, updated.IsDefault)

		// Verify first is no longer default
		firstUpdated, err := addressSvc.GetAddress(t.Context(), first.ID, freshCustomerID)
		require.NoError(t, err)
		assert.False(t, firstUpdated.IsDefault)
	})

	t.Run("update with stale version returns ErrInvalidVersion", func(t *testing.T) {
		created, err := addressSvc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{
			CustomerID: testCustomerID,
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.NoError(t, err)

		// First update
		_, err = addressSvc.UpdateAddress(t.Context(), addressapp.UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: testCustomerID,
			Label:      "Work",
			FullName:   "Jane Smith",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Version:    created.Version,
		})
		require.NoError(t, err)

		// Second update with stale version
		_, err = addressSvc.UpdateAddress(t.Context(), addressapp.UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: testCustomerID,
			Label:      "Work2",
			FullName:   "Jane Smith",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Version:    created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("delete with wrong customer ID returns ErrAddressNotFound", func(t *testing.T) {
		// Create another customer
		otherCustomer, err := customerSvc.CreateCustomer(t.Context(), customerapp.CreateCustomerRequest{
			UserID:      "test-user-" + uniqueID(),
			DisplayName: "Other User",
		})
		require.NoError(t, err)

		created, err := addressSvc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{
			CustomerID: testCustomerID,
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.NoError(t, err)

		err = addressSvc.DeleteAddress(t.Context(), addressapp.DeleteAddressRequest{
			ID:         created.ID,
			CustomerID: otherCustomer.ID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("get with wrong customer ID returns ErrAddressNotFound", func(t *testing.T) {
		created, err := addressSvc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{
			CustomerID: testCustomerID,
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.NoError(t, err)

		_, err = addressSvc.GetAddress(t.Context(), created.ID, "wrong-customer-id")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("update with wrong customer ID returns ErrAddressNotFound", func(t *testing.T) {
		created, err := addressSvc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{
			CustomerID: testCustomerID,
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.NoError(t, err)

		_, err = addressSvc.UpdateAddress(t.Context(), addressapp.UpdateAddressRequest{
			ID:         created.ID,
			CustomerID: "wrong-customer-id",
			Label:      "Work",
			FullName:   "Jane Smith",
			Street:     "456 Office Ave",
			City:       "San Francisco",
			State:      "CA",
			Zip:        "94102",
			Country:    "USA",
			Version:    created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("set default with wrong customer ID returns ErrAddressNotFound", func(t *testing.T) {
		created, err := addressSvc.CreateAddress(t.Context(), addressapp.CreateAddressRequest{
			CustomerID: testCustomerID,
			Label:      "Home",
			FullName:   "John Doe",
			Street:     "123 Main St",
			City:       "New York",
			State:      "NY",
			Zip:        "10001",
			Country:    "USA",
		})
		require.NoError(t, err)

		_, err = addressSvc.SetDefaultAddress(t.Context(), addressapp.SetDefaultAddressRequest{
			ID:         created.ID,
			CustomerID: "wrong-customer-id",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})
}

func uniqueID() string {
	return "test-" + time.Now().Format("20060102150405.000000000")
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
