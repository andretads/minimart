//go:build integration

package test

import (
	"minimart/internal/customer/domain"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCustomerIntegration(t *testing.T) {
	svc := newCustomerServices(t)

	t.Run("Blank user_id is rejected", func(t *testing.T) {
		req := validCreateCustomerRequest()
		req.UserID = ""

		_, err := svc.CreateCustomer(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Display name too long is rejected", func(t *testing.T) {
		req := validCreateCustomerRequest()
		req.UserID = newTestUserID()
		req.DisplayName = strings.Repeat("a", 201)

		_, err := svc.CreateCustomer(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrDisplayNameTooLong)
	})

	t.Run("Customer got created with no problems", func(t *testing.T) {
		req := validCreateCustomerRequest()
		req.UserID = newTestUserID()
		req.DisplayName = "John"

		created := createTestCustomer(t, svc, req)
		stored := getTestCustomer(t, svc, created.ID)

		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, req.UserID, stored.UserID)
		assert.Equal(t, "John", stored.DisplayName)
		assert.Equal(t, domain.StatusActive, stored.Status)
		assert.False(t, stored.CreatedAt.IsZero())
	})

	t.Run("Customer without display name gets created", func(t *testing.T) {
		req := validCreateCustomerRequest()
		req.UserID = newTestUserID()
		req.DisplayName = ""

		created := createTestCustomer(t, svc, req)
		stored := getTestCustomer(t, svc, created.ID)

		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, req.UserID, stored.UserID)
		assert.Empty(t, stored.DisplayName)
		assert.Equal(t, domain.StatusActive, stored.Status)
		assert.False(t, stored.CreatedAt.IsZero())
	})

	t.Run("Duplicate user_id is rejected", func(t *testing.T) {
		userID := newTestUserID()
		req := validCreateCustomerRequest()
		req.UserID = userID
		_, err := svc.CreateCustomer(t.Context(), req)
		require.NoError(t, err)

		req2 := validCreateCustomerRequest()
		req2.UserID = userID
		req2.DisplayName = "Jane"
		_, err = svc.CreateCustomer(t.Context(), req2)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerExists)
	})
}
