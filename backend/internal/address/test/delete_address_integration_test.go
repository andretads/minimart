//go:build integration

package test

import (
	"minimart/internal/address/domain"
	"testing"

	addressapp "minimart/internal/address/application"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteAddressIntegration(t *testing.T) {
	svc, customerSvc := newAddressServices(t)

	t.Run("Empty customer ID is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		err := svc.DeleteAddress(t.Context(), addressapp.DeleteAddressRequest{
			ID:         addr.ID,
			CustomerID: "",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("Unknown ID is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)

		err := svc.DeleteAddress(t.Context(), addressapp.DeleteAddressRequest{
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

		err := svc.DeleteAddress(t.Context(), addressapp.DeleteAddressRequest{
			ID:         addr.ID,
			CustomerID: otherID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Address deleted", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		err := svc.DeleteAddress(t.Context(), addressapp.DeleteAddressRequest{
			ID:         addr.ID,
			CustomerID: customerID,
		})
		require.NoError(t, err)

		_, err = svc.GetAddress(t.Context(), addr.ID, customerID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Deleted address no longer retrievable", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(customerID))

		err := svc.DeleteAddress(t.Context(), addressapp.DeleteAddressRequest{
			ID:         addr.ID,
			CustomerID: customerID,
		})
		require.NoError(t, err)

		_, err = svc.GetAddress(t.Context(), addr.ID, customerID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Deleted address no longer appears in list", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)

		homeReq := validCreateAddressRequest(customerID)
		homeReq.Label = "Home"
		home := createTestAddress(t, svc, homeReq)

		workReq := validCreateAddressRequest(customerID)
		workReq.Label = "Work"
		createTestAddress(t, svc, workReq)

		err := svc.DeleteAddress(t.Context(), addressapp.DeleteAddressRequest{
			ID:         home.ID,
			CustomerID: customerID,
		})
		require.NoError(t, err)

		addresses, err := svc.ListAddresses(t.Context(), addressapp.ListAddressesRequest{
			CustomerID: customerID,
		})
		require.NoError(t, err)
		require.Len(t, addresses, 1)
		assert.Equal(t, "Work", addresses[0].Label)
	})
}
