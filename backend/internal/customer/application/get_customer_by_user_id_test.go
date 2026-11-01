package application

import (
	"minimart/internal/customer/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCustomerByUserID(t *testing.T) {
	t.Run("Unknown user_id is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.GetCustomerByUserID(t.Context(), "unknown")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})

	t.Run("Customer resolved by user_id", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateCustomerRequest()
		req.UserID = "user-1"
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		got, err := svc.GetCustomerByUserID(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "user-1", got.UserID)
		assert.Equal(t, "John", got.DisplayName)
		assert.Equal(t, domain.StatusActive, got.Status)
	})
}
