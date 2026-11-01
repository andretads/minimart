//go:build integration

package test

import (
	"minimart/internal/customer/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureCustomerIntegration(t *testing.T) {
	svc := newCustomerServices(t)

	t.Run("Empty user_id is rejected", func(t *testing.T) {
		_, err := svc.EnsureCustomer(t.Context(), "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Existing customer returned unchanged", func(t *testing.T) {
		userID := newTestUserID()
		req := validCreateCustomerRequest()
		req.UserID = userID
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		got, err := svc.EnsureCustomer(t.Context(), userID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, userID, got.UserID)
		assert.Equal(t, "John", got.DisplayName)
		assert.Equal(t, domain.StatusActive, got.Status)
	})

	t.Run("Missing customer created with active status and empty display name", func(t *testing.T) {
		userID := newTestUserID()

		got, err := svc.EnsureCustomer(t.Context(), userID)
		require.NoError(t, err)
		assert.NotEmpty(t, got.ID)
		assert.Equal(t, userID, got.UserID)
		assert.Empty(t, got.DisplayName)
		assert.Equal(t, domain.StatusActive, got.Status)
		assert.Equal(t, 1, got.Version)
	})

	t.Run("Concurrent create for same user_id returns the same customer", func(t *testing.T) {
		userID := newTestUserID()

		c1, err := svc.EnsureCustomer(t.Context(), userID)
		require.NoError(t, err)
		c2, err := svc.EnsureCustomer(t.Context(), userID)
		require.NoError(t, err)

		assert.Equal(t, c1.ID, c2.ID)

		stored := getTestCustomer(t, svc, c1.ID)
		assert.Equal(t, userID, stored.UserID)
	})
}
