//go:build integration

package test

import (
	"minimart/internal/address/domain"
	"testing"

	addressapp "minimart/internal/address/application"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetDefaultAddressIntegration(t *testing.T) {
	svc, customerSvc := newAddressServices(t)

	t.Run("Empty customer ID is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		_, err := svc.SetDefaultAddress(t.Context(), addressapp.SetDefaultAddressRequest{
			ID:         addr.ID,
			CustomerID: "",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("Unknown ID is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)

		_, err := svc.SetDefaultAddress(t.Context(), addressapp.SetDefaultAddressRequest{
			ID:         "unknown",
			CustomerID: customerID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Address owned by another customer is not found", func(t *testing.T) {
		ownerID := newTestCustomer(t, customerSvc)
		otherID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(ownerID))

		_, err := svc.SetDefaultAddress(t.Context(), addressapp.SetDefaultAddressRequest{
			ID:         addr.ID,
			CustomerID: otherID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Address set as default", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		req := validCreateAddressRequest(customerID)
		req.IsDefault = false
		addr := createTestAddress(t, svc, req)

		updated, err := svc.SetDefaultAddress(t.Context(), addressapp.SetDefaultAddressRequest{
			ID:         addr.ID,
			CustomerID: customerID,
		})
		require.NoError(t, err)

		stored := getTestAddress(t, svc, updated.ID, customerID)
		assert.Equal(t, addr.ID, stored.ID)
		assert.True(t, stored.IsDefault)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Version is bumped on set default", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		req := validCreateAddressRequest(customerID)
		req.IsDefault = false
		addr := createTestAddress(t, svc, req)

		_, err := svc.SetDefaultAddress(t.Context(), addressapp.SetDefaultAddressRequest{
			ID:         addr.ID,
			CustomerID: customerID,
		})
		require.NoError(t, err)

		stored := getTestAddress(t, svc, addr.ID, customerID)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Previous default is unset", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)

		firstReq := validCreateAddressRequest(customerID)
		firstReq.IsDefault = true
		first := createTestAddress(t, svc, firstReq)

		secondReq := validCreateAddressRequest(customerID)
		secondReq.IsDefault = false
		second := createTestAddress(t, svc, secondReq)

		_, err := svc.SetDefaultAddress(t.Context(), addressapp.SetDefaultAddressRequest{
			ID:         second.ID,
			CustomerID: customerID,
		})
		require.NoError(t, err)

		storedSecond := getTestAddress(t, svc, second.ID, customerID)
		assert.True(t, storedSecond.IsDefault)

		storedFirst := getTestAddress(t, svc, first.ID, customerID)
		assert.False(t, storedFirst.IsDefault)
	})

	t.Run("Setting an already default address keeps it default", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		req := validCreateAddressRequest(customerID)
		req.IsDefault = true
		addr := createTestAddress(t, svc, req)

		updated, err := svc.SetDefaultAddress(t.Context(), addressapp.SetDefaultAddressRequest{
			ID:         addr.ID,
			CustomerID: customerID,
		})
		require.NoError(t, err)

		stored := getTestAddress(t, svc, updated.ID, customerID)
		assert.Equal(t, addr.ID, stored.ID)
		assert.True(t, stored.IsDefault)
	})
}
