//go:build integration

package test

import (
	"minimart/internal/address/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAddressIntegration(t *testing.T) {
	svc, customerSvc := newAddressServices(t)

	t.Run("Empty customer ID is rejected", func(t *testing.T) {
		_, err := svc.GetAddress(t.Context(), "addr-1", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyCustomerID)
	})

	t.Run("Unknown ID is rejected", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)

		_, err := svc.GetAddress(t.Context(), "unknown", customerID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Address owned by another customer is not found", func(t *testing.T) {
		ownerID := newTestCustomer(t, customerSvc)
		otherID := newTestCustomer(t, customerSvc)
		addr := createTestAddress(t, svc, validCreateAddressRequest(ownerID))

		_, err := svc.GetAddress(t.Context(), addr.ID, otherID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAddressNotFound)
	})

	t.Run("Address retrieved by ID", func(t *testing.T) {
		customerID := newTestCustomer(t, customerSvc)
		req := validCreateAddressRequest(customerID)
		req.Label = "Home"
		req.FullName = "John Doe"
		created := createTestAddress(t, svc, req)

		got, err := svc.GetAddress(t.Context(), created.ID, customerID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Home", got.Label)
		assert.Equal(t, "John Doe", got.FullName)
		assert.Equal(t, 1, got.Version)
		assert.False(t, got.CreatedAt.IsZero())
	})
}
