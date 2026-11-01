package module

import (
	"errors"
	"minimart/internal/address/application"
	"minimart/internal/address/domain"
	"minimart/internal/address/infrastructure/memory"
	"minimart/internal/address/presentation/module/client"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultAddressByCustomerID(t *testing.T) {
	addr, err := domain.NewAddress("addr-1", "cust-1", domain.AddressFields{Label: "Home", FullName: "John Doe", Street: "123 Main St", City: "New York", State: "NY", Zip: "10001", Country: "USA"}, true)
	require.NoError(t, err)

	t.Run("maps a domain address into the client type", func(t *testing.T) {
		repo := memory.New()
		require.NoError(t, repo.Save(t.Context(), addr))
		_, err := repo.SetDefault(t.Context(), "addr-1", "cust-1")
		require.NoError(t, err)
		adapter := New(application.NewService(repo, metrics.Noop{}))

		got, err := adapter.GetDefaultAddressByCustomerID(t.Context(), "cust-1")
		require.NoError(t, err)
		assert.Equal(t, "addr-1", got.ID)
		assert.Equal(t, "cust-1", got.CustomerID)
		assert.Equal(t, "Home", got.Label)
		assert.Equal(t, "John Doe", got.FullName)
	})

	t.Run("translates not-found into the client sentinel", func(t *testing.T) {
		adapter := New(application.NewService(memory.New(), metrics.Noop{}))

		_, err := adapter.GetDefaultAddressByCustomerID(t.Context(), "cust-1")
		assert.ErrorIs(t, err, client.ErrAddressNotFound)
	})

	t.Run("passes through unexpected errors unchanged", func(t *testing.T) {
		boom := errors.New("boom")
		failing := failingRepo{err: boom}
		adapter := New(application.NewService(failing, metrics.Noop{}))

		_, err := adapter.GetDefaultAddressByCustomerID(t.Context(), "cust-1")
		require.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, client.ErrAddressNotFound)
	})

	t.Run("empty customer id is rejected", func(t *testing.T) {
		adapter := New(application.NewService(memory.New(), metrics.Noop{}))

		_, err := adapter.GetDefaultAddressByCustomerID(t.Context(), "")
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})
}
