package application

import (
	"minimart/internal/address/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteAddress(t *testing.T) {
	t.Run("Empty customer ID is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)
		addr := createTestAddress(t, svc, validCreateAddressRequest("cust-1"))

		err := svc.DeleteAddress(t.Context(), DeleteAddressRequest{
			ID:         addr.ID,
			CustomerID: "",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("Unknown ID is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		err := svc.DeleteAddress(t.Context(), DeleteAddressRequest{
			ID:         "unknown",
			CustomerID: "cust-1",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Address owned by another customer is not found", func(t *testing.T) {
		svc, _ := newTestService(t)
		addr := createTestAddress(t, svc, validCreateAddressRequest("cust-1"))

		err := svc.DeleteAddress(t.Context(), DeleteAddressRequest{
			ID:         addr.ID,
			CustomerID: "cust-2",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Address deleted", func(t *testing.T) {
		svc, _ := newTestService(t)
		addr := createTestAddress(t, svc, validCreateAddressRequest("cust-1"))

		err := svc.DeleteAddress(t.Context(), DeleteAddressRequest{
			ID:         addr.ID,
			CustomerID: "cust-1",
		})
		require.NoError(t, err)

		_, err = svc.GetAddress(t.Context(), addr.ID, "cust-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Deleted address no longer retrievable", func(t *testing.T) {
		svc, _ := newTestService(t)
		addr := createTestAddress(t, svc, validCreateAddressRequest("cust-1"))

		err := svc.DeleteAddress(t.Context(), DeleteAddressRequest{
			ID:         addr.ID,
			CustomerID: "cust-1",
		})
		require.NoError(t, err)

		_, err = svc.GetAddress(t.Context(), addr.ID, "cust-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Deleted address no longer appears in list", func(t *testing.T) {
		svc, _ := newTestService(t)

		homeReq := validCreateAddressRequest("cust-1")
		homeReq.Label = "Home"
		home := createTestAddress(t, svc, homeReq)

		workReq := validCreateAddressRequest("cust-1")
		workReq.Label = "Work"
		createTestAddress(t, svc, workReq)

		err := svc.DeleteAddress(t.Context(), DeleteAddressRequest{
			ID:         home.ID,
			CustomerID: "cust-1",
		})
		require.NoError(t, err)

		addresses, err := svc.ListAddresses(t.Context(), ListAddressesRequest{
			CustomerID: "cust-1",
		})
		require.NoError(t, err)
		require.Len(t, addresses, 1)
		assert.Equal(t, "Work", addresses[0].Label)
	})
}
