//go:build integration

package test

import (
	"minimart/internal/customer/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeactivateCustomerIntegration(t *testing.T) {
	svc := newCustomerServices(t)

	t.Run("Unknown ID is rejected", func(t *testing.T) {
		_, err := svc.DeactivateCustomer(t.Context(), "unknown")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})

	t.Run("Customer deactivated", func(t *testing.T) {
		userID := newTestUserID()
		req := validCreateCustomerRequest()
		req.UserID = userID
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		deactivated, err := svc.DeactivateCustomer(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, userID, deactivated.UserID)
		assert.Equal(t, domain.StatusDeactivated, deactivated.Status)
		assert.NotNil(t, deactivated.DeactivatedAt)
		assert.Equal(t, 2, deactivated.Version)

		stored := getTestCustomer(t, svc, created.ID)
		assert.Equal(t, domain.StatusDeactivated, stored.Status)
		assert.NotNil(t, stored.DeactivatedAt)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Deactivating already deactivated is idempotent", func(t *testing.T) {
		userID := newTestUserID()
		req := validCreateCustomerRequest()
		req.UserID = userID
		created := createTestCustomer(t, svc, req)

		deact1, err := svc.DeactivateCustomer(t.Context(), created.ID)
		require.NoError(t, err)

		deact2, err := svc.DeactivateCustomer(t.Context(), created.ID)
		require.NoError(t, err)

		assert.Equal(t, domain.StatusDeactivated, deact2.Status)
		assert.Equal(t, deact1.DeactivatedAt, deact2.DeactivatedAt)
		assert.Equal(t, 3, deact2.Version)
	})
}
