package application

import (
	"minimart/internal/customer/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeactivateCustomer(t *testing.T) {
	t.Run("Unknown ID is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.DeactivateCustomer(t.Context(), "unknown")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})

	t.Run("Customer deactivated", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateCustomerRequest()
		req.UserID = "user-1"
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		deactivated, err := svc.DeactivateCustomer(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "user-1", deactivated.UserID)
		assert.Equal(t, domain.StatusDeactivated, deactivated.Status)
		assert.NotNil(t, deactivated.DeactivatedAt)
		assert.Equal(t, 2, deactivated.Version)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusDeactivated, stored.Status)
		assert.NotNil(t, stored.DeactivatedAt)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Deactivating already deactivated is idempotent", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateCustomerRequest()
		req.UserID = "user-1"
		created := createTestCustomer(t, svc, req)

		deact1, err := svc.DeactivateCustomer(t.Context(), created.ID)
		require.NoError(t, err)

		deact2, err := svc.DeactivateCustomer(t.Context(), created.ID)
		require.NoError(t, err)

		assert.Equal(t, domain.StatusDeactivated, deact2.Status)
		assert.Equal(t, deact1.DeactivatedAt, deact2.DeactivatedAt)
		assert.Equal(t, 3, deact2.Version)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusDeactivated, stored.Status)
		assert.NotNil(t, stored.DeactivatedAt)
	})
}
