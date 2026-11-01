//go:build integration

package test

import (
	"minimart/internal/address/domain"
	"testing"

	addressapp "minimart/internal/address/application"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAddressesIntegration(t *testing.T) {
	svc, customerSvc := newAddressServices(t)

	t.Run("Empty customer ID is rejected", func(t *testing.T) {
		_, err := svc.ListAddresses(t.Context(), addressapp.ListAddressesRequest{
			CustomerID: "",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("Empty storage returns an empty list", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)

		addresses, err := svc.ListAddresses(t.Context(), addressapp.ListAddressesRequest{
			CustomerID: customerID,
		})
		require.NoError(t, err)
		assert.Empty(t, addresses)
	})

	t.Run("All addresses for the customer are returned", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)

		homeReq := validCreateAddressRequest(customerID)
		homeReq.Label = "Home"
		createTestAddress(t, svc, homeReq)

		workReq := validCreateAddressRequest(customerID)
		workReq.Label = "Work"
		createTestAddress(t, svc, workReq)

		addresses, err := svc.ListAddresses(t.Context(), addressapp.ListAddressesRequest{
			CustomerID: customerID,
		})
		require.NoError(t, err)
		assert.Len(t, addresses, 2)
	})

	t.Run("Default address comes first, then by created_at descending", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)

		workReq := validCreateAddressRequest(customerID)
		workReq.Label = "Work"
		workReq.IsDefault = false
		createTestAddress(t, svc, workReq)

		homeReq := validCreateAddressRequest(customerID)
		homeReq.Label = "Home"
		homeReq.IsDefault = true
		createTestAddress(t, svc, homeReq)

		otherReq := validCreateAddressRequest(customerID)
		otherReq.Label = "Other"
		otherReq.IsDefault = false
		createTestAddress(t, svc, otherReq)

		addresses, err := svc.ListAddresses(t.Context(), addressapp.ListAddressesRequest{
			CustomerID: customerID,
		})
		require.NoError(t, err)
		require.Len(t, addresses, 3)
		assert.Equal(t, "Home", addresses[0].Label)
		assert.True(t, addresses[0].IsDefault)
		assert.Equal(t, "Other", addresses[1].Label)
		assert.Equal(t, "Work", addresses[2].Label)
	})

	t.Run("Addresses of other customers are not included", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		otherID := newTestCustomer(t, customerSvc)

		homeReq := validCreateAddressRequest(customerID)
		homeReq.Label = "Home"
		createTestAddress(t, svc, homeReq)

		workReq := validCreateAddressRequest(otherID)
		workReq.Label = "Work"
		createTestAddress(t, svc, workReq)

		addresses, err := svc.ListAddresses(t.Context(), addressapp.ListAddressesRequest{
			CustomerID: customerID,
		})
		require.NoError(t, err)
		require.Len(t, addresses, 1)
		assert.Equal(t, "Home", addresses[0].Label)
	})
}
