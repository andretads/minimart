package application

import (
	"minimart/internal/address/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAddresses(t *testing.T) {
	t.Run("Empty customer ID is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.ListAddresses(t.Context(), ListAddressesRequest{
			CustomerID: "",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("Empty storage returns an empty list", func(t *testing.T) {
		svc, _ := newTestService(t)

		addresses, err := svc.ListAddresses(t.Context(), ListAddressesRequest{
			CustomerID: "cust-1",
		})
		require.NoError(t, err)
		assert.Empty(t, addresses)
	})

	t.Run("All addresses for the customer are returned", func(t *testing.T) {
		svc, _ := newTestService(t)

		homeReq := validCreateAddressRequest("cust-1")
		homeReq.Label = "Home"
		createTestAddress(t, svc, homeReq)

		workReq := validCreateAddressRequest("cust-1")
		workReq.Label = "Work"
		createTestAddress(t, svc, workReq)

		addresses, err := svc.ListAddresses(t.Context(), ListAddressesRequest{
			CustomerID: "cust-1",
		})
		require.NoError(t, err)
		assert.Len(t, addresses, 2)
	})

	t.Run("Default address comes first, then by created_at descending", func(t *testing.T) {
		svc, _ := newTestService(t)

		workReq := validCreateAddressRequest("cust-1")
		workReq.Label = "Work"
		workReq.IsDefault = false
		createTestAddress(t, svc, workReq)

		homeReq := validCreateAddressRequest("cust-1")
		homeReq.Label = "Home"
		homeReq.IsDefault = true
		createTestAddress(t, svc, homeReq)

		otherReq := validCreateAddressRequest("cust-1")
		otherReq.Label = "Other"
		otherReq.IsDefault = false
		createTestAddress(t, svc, otherReq)

		addresses, err := svc.ListAddresses(t.Context(), ListAddressesRequest{
			CustomerID: "cust-1",
		})
		require.NoError(t, err)
		require.Len(t, addresses, 3)
		assert.Equal(t, "Home", addresses[0].Label)
		assert.True(t, addresses[0].IsDefault)
		assert.Equal(t, "Other", addresses[1].Label)
		assert.Equal(t, "Work", addresses[2].Label)
	})

	t.Run("Addresses of other customers are not included", func(t *testing.T) {
		svc, _ := newTestService(t)

		homeReq := validCreateAddressRequest("cust-1")
		homeReq.Label = "Home"
		createTestAddress(t, svc, homeReq)

		workReq := validCreateAddressRequest("cust-2")
		workReq.Label = "Work"
		createTestAddress(t, svc, workReq)

		addresses, err := svc.ListAddresses(t.Context(), ListAddressesRequest{
			CustomerID: "cust-1",
		})
		require.NoError(t, err)
		require.Len(t, addresses, 1)
		assert.Equal(t, "Home", addresses[0].Label)
	})
}
