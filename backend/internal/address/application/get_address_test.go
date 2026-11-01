package application

import (
	"minimart/internal/address/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAddress(t *testing.T) {
	t.Run("Empty customer ID is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.GetAddress(t.Context(), "addr-1", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("Unknown ID is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.GetAddress(t.Context(), "unknown", "cust-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Address owned by another customer is not found", func(t *testing.T) {
		svc, _ := newTestService(t)
		addr := createTestAddress(t, svc, validCreateAddressRequest("cust-1"))

		_, err := svc.GetAddress(t.Context(), addr.ID, "cust-2")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Address retrieved by ID", func(t *testing.T) {
		svc, _ := newTestService(t)
		req := validCreateAddressRequest("cust-1")
		req.Label = "Home"
		req.FullName = "John Doe"
		addr := createTestAddress(t, svc, req)

		got, err := svc.GetAddress(t.Context(), addr.ID, "cust-1")
		require.NoError(t, err)
		assert.Equal(t, addr.ID, got.ID)
		assert.Equal(t, "Home", got.Label)
		assert.Equal(t, "John Doe", got.FullName)
		assert.Equal(t, 1, got.Version)
		assert.False(t, got.CreatedAt.IsZero())
	})
}
