//go:build integration

package test

import (
	"minimart/internal/customer/application"
	"minimart/internal/customer/domain"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCustomerProfileIntegration(t *testing.T) {
	svc := newCustomerServices(t)

	t.Run("Display name too long is rejected", func(t *testing.T) {
		userID := newTestUserID()
		req := validCreateCustomerRequest()
		req.UserID = userID
		created := createTestCustomer(t, svc, req)

		_, err := svc.UpdateCustomerProfile(t.Context(), application.UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: strings.Repeat("a", 201),
			Version:     created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrDisplayNameTooLong)
	})

	t.Run("Unknown ID is rejected", func(t *testing.T) {
		_, err := svc.UpdateCustomerProfile(t.Context(), application.UpdateCustomerProfileRequest{
			ID:          "unknown",
			DisplayName: "John",
			Version:     1,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})

	t.Run("Missing version is rejected", func(t *testing.T) {
		userID := newTestUserID()
		req := validCreateCustomerRequest()
		req.UserID = userID
		created := createTestCustomer(t, svc, req)

		_, err := svc.UpdateCustomerProfile(t.Context(), application.UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     0,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("Stale version is rejected", func(t *testing.T) {
		userID := newTestUserID()
		req := validCreateCustomerRequest()
		req.UserID = userID
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		_, err := svc.UpdateCustomerProfile(t.Context(), application.UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     created.Version,
		})
		require.NoError(t, err)

		_, err = svc.UpdateCustomerProfile(t.Context(), application.UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane2",
			Version:     created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerOptimisticLock)
	})

	t.Run("Stale ETag is rejected", func(t *testing.T) {
		userID := newTestUserID()
		req := validCreateCustomerRequest()
		req.UserID = userID
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		_, err := svc.UpdateCustomerProfile(t.Context(), application.UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     99,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerOptimisticLock)
	})

	t.Run("Customer display name updated", func(t *testing.T) {
		userID := newTestUserID()
		req := validCreateCustomerRequest()
		req.UserID = userID
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		updated, err := svc.UpdateCustomerProfile(t.Context(), application.UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     created.Version,
		})
		require.NoError(t, err)
		assert.Equal(t, userID, updated.UserID)
		assert.Equal(t, "Jane", updated.DisplayName)
		assert.Equal(t, domain.StatusActive, updated.Status)
		assert.Equal(t, created.Version+1, updated.Version)

		stored := getTestCustomer(t, svc, created.ID)
		assert.Equal(t, "Jane", stored.DisplayName)
	})

	t.Run("Customer is updated using the ETag from GET", func(t *testing.T) {
		userID := newTestUserID()
		req := validCreateCustomerRequest()
		req.UserID = userID
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		updated, err := svc.UpdateCustomerProfile(t.Context(), application.UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     1,
		})
		require.NoError(t, err)
		assert.Equal(t, userID, updated.UserID)
		assert.Equal(t, "Jane", updated.DisplayName)
		assert.Equal(t, domain.StatusActive, updated.Status)
		assert.Equal(t, 2, updated.Version)

		stored := getTestCustomer(t, svc, created.ID)
		assert.Equal(t, "Jane", stored.DisplayName)
	})

	t.Run("Display name cleared", func(t *testing.T) {
		userID := newTestUserID()
		req := validCreateCustomerRequest()
		req.UserID = userID
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		updated, err := svc.UpdateCustomerProfile(t.Context(), application.UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "",
			Version:     created.Version,
		})
		require.NoError(t, err)
		assert.Equal(t, userID, updated.UserID)
		assert.Empty(t, updated.DisplayName)
		assert.Equal(t, domain.StatusActive, updated.Status)
		assert.Equal(t, created.Version+1, updated.Version)

		stored := getTestCustomer(t, svc, created.ID)
		assert.Empty(t, stored.DisplayName)
	})
}
