package application

import (
	"minimart/internal/address/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultAddress(t *testing.T) {
	t.Run("Empty customer ID is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.GetDefaultAddress(t.Context(), "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("No default address exists", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateAddressRequest("cust-1")
		req.IsDefault = false
		createTestAddress(t, svc, req)

		_, err := svc.GetDefaultAddress(t.Context(), "cust-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Default address returned", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateAddressRequest("cust-1")
		req.Label = "Home"
		req.IsDefault = true
		addr := createTestAddress(t, svc, req)

		got, err := svc.GetDefaultAddress(t.Context(), "cust-1")
		require.NoError(t, err)
		assert.Equal(t, addr.ID, got.ID)
		assert.Equal(t, "Home", got.Label)
		assert.True(t, got.IsDefault)
	})
}
