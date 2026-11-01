package application

import (
	"minimart/internal/address/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetDefaultAddress(t *testing.T) {
	t.Run("Empty customer ID is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)
		addr := createTestAddress(t, svc, validCreateAddressRequest("cust-1"))

		_, err := svc.SetDefaultAddress(t.Context(), SetDefaultAddressRequest{
			ID:         addr.ID,
			CustomerID: "",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("Unknown ID is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.SetDefaultAddress(t.Context(), SetDefaultAddressRequest{
			ID:         "unknown",
			CustomerID: "cust-1",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Address owned by another customer is not found", func(t *testing.T) {
		svc, _ := newTestService(t)
		addr := createTestAddress(t, svc, validCreateAddressRequest("cust-1"))

		_, err := svc.SetDefaultAddress(t.Context(), SetDefaultAddressRequest{
			ID:         addr.ID,
			CustomerID: "cust-2",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Address set as default", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateAddressRequest("cust-1")
		req.IsDefault = false
		addr := createTestAddress(t, svc, req)

		updated, err := svc.SetDefaultAddress(t.Context(), SetDefaultAddressRequest{
			ID:         addr.ID,
			CustomerID: "cust-1",
		})
		require.NoError(t, err)

		stored, err := svc.GetAddress(t.Context(), updated.ID, "cust-1")
		require.NoError(t, err)
		assert.Equal(t, addr.ID, stored.ID)
		assert.True(t, stored.IsDefault)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Version is bumped on set default", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateAddressRequest("cust-1")
		req.IsDefault = false
		addr := createTestAddress(t, svc, req)

		_, err := svc.SetDefaultAddress(t.Context(), SetDefaultAddressRequest{
			ID:         addr.ID,
			CustomerID: "cust-1",
		})
		require.NoError(t, err)

		stored, err := svc.GetAddress(t.Context(), addr.ID, "cust-1")
		require.NoError(t, err)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Previous default is unset", func(t *testing.T) {
		svc, _ := newTestService(t)

		firstReq := validCreateAddressRequest("cust-1")
		firstReq.IsDefault = true
		first := createTestAddress(t, svc, firstReq)

		secondReq := validCreateAddressRequest("cust-1")
		secondReq.IsDefault = false
		second := createTestAddress(t, svc, secondReq)

		_, err := svc.SetDefaultAddress(t.Context(), SetDefaultAddressRequest{
			ID:         second.ID,
			CustomerID: "cust-1",
		})
		require.NoError(t, err)

		storedSecond, err := svc.GetAddress(t.Context(), second.ID, "cust-1")
		require.NoError(t, err)
		assert.True(t, storedSecond.IsDefault)

		storedFirst, err := svc.GetAddress(t.Context(), first.ID, "cust-1")
		require.NoError(t, err)
		assert.False(t, storedFirst.IsDefault)
	})

	t.Run("Setting an already default address keeps it default", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateAddressRequest("cust-1")
		req.IsDefault = true
		addr := createTestAddress(t, svc, req)

		updated, err := svc.SetDefaultAddress(t.Context(), SetDefaultAddressRequest{
			ID:         addr.ID,
			CustomerID: "cust-1",
		})
		require.NoError(t, err)

		stored, err := svc.GetAddress(t.Context(), updated.ID, "cust-1")
		require.NoError(t, err)
		assert.Equal(t, addr.ID, stored.ID)
		assert.True(t, stored.IsDefault)
	})
}
