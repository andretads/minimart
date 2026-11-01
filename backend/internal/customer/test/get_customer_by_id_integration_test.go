//go:build integration

package test

import (
	"minimart/internal/customer/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCustomerByIDIntegration(t *testing.T) {
	svc := newCustomerServices(t)

	t.Run("Unknown ID is rejected", func(t *testing.T) {
		_, err := svc.GetCustomerByID(t.Context(), "unknown")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})

	t.Run("Customer retrieved by ID", func(t *testing.T) {
		userID := newTestUserID()
		req := validCreateCustomerRequest()
		req.UserID = userID
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		got, err := svc.GetCustomerByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, userID, got.UserID)
		assert.Equal(t, "John", got.DisplayName)
		assert.Equal(t, domain.StatusActive, got.Status)
	})
}
