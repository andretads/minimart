//go:build integration

package test

import (
	"minimart/internal/customer/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCustomerByUserIDIntegration(t *testing.T) {
	svc := newCustomerServices(t)

	t.Run("Unknown user_id is rejected", func(t *testing.T) {
		_, err := svc.GetCustomerByUserID(t.Context(), "unknown-"+newTestUserID())
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})

	t.Run("Customer resolved by user_id", func(t *testing.T) {
		userID := newTestUserID()
		req := validCreateCustomerRequest()
		req.UserID = userID
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		got, err := svc.GetCustomerByUserID(t.Context(), userID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, userID, got.UserID)
		assert.Equal(t, "John", got.DisplayName)
		assert.Equal(t, domain.StatusActive, got.Status)
	})
}
