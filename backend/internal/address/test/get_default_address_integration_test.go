//go:build integration

package test

import (
	"minimart/internal/address/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultAddressIntegration(t *testing.T) {
	svc, customerSvc := newAddressServices(t)

	t.Run("Empty customer ID is rejected", func(t *testing.T) {
		_, err := svc.GetDefaultAddress(t.Context(), "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("No default address exists", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		req := validCreateAddressRequest(customerID)
		req.IsDefault = false
		createTestAddress(t, svc, req)

		_, err := svc.GetDefaultAddress(t.Context(), customerID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Default address returned", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		req := validCreateAddressRequest(customerID)
		req.Label = "Home"
		req.IsDefault = true
		created := createTestAddress(t, svc, req)

		got, err := svc.GetDefaultAddress(t.Context(), customerID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Home", got.Label)
		assert.True(t, got.IsDefault)
	})
}
