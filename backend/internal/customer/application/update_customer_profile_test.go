package application

import (
	"minimart/internal/customer/domain"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCustomerProfile(t *testing.T) {
	t.Run("Display name too long is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateCustomerRequest()
		req.UserID = "user-1"
		created := createTestCustomer(t, svc, req)

		_, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: strings.Repeat("a", 201),
			Version:     created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrDisplayNameTooLong)
	})

	t.Run("Unknown ID is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          "unknown",
			DisplayName: "John",
			Version:     1,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerNotFound)
	})

	t.Run("Missing version is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateCustomerRequest()
		req.UserID = "user-1"
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		_, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     0,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidVersion)
	})

	t.Run("Stale version is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateCustomerRequest()
		req.UserID = "user-1"
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		// another writer updates the customer
		_, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     created.Version,
		})
		require.NoError(t, err)

		// stale copy update
		_, err = svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane2",
			Version:     created.Version,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerOptimisticLock)
	})

	t.Run("Stale ETag is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateCustomerRequest()
		req.UserID = "user-1"
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		_, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     99,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerOptimisticLock)
	})

	t.Run("Customer display name updated", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateCustomerRequest()
		req.UserID = "user-1"
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		updated, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     created.Version,
		})
		require.NoError(t, err)
		assert.Equal(t, "user-1", updated.UserID)
		assert.Equal(t, "Jane", updated.DisplayName)
		assert.Equal(t, domain.StatusActive, updated.Status)
		assert.Equal(t, created.Version+1, updated.Version)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Jane", stored.DisplayName)
	})

	t.Run("Customer is updated using the ETag from GET", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateCustomerRequest()
		req.UserID = "user-1"
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		updated, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "Jane",
			Version:     1,
		})
		require.NoError(t, err)
		assert.Equal(t, "user-1", updated.UserID)
		assert.Equal(t, "Jane", updated.DisplayName)
		assert.Equal(t, domain.StatusActive, updated.Status)
		assert.Equal(t, 2, updated.Version)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Jane", stored.DisplayName)
	})

	t.Run("Display name cleared", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateCustomerRequest()
		req.UserID = "user-1"
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		updated, err := svc.UpdateCustomerProfile(t.Context(), UpdateCustomerProfileRequest{
			ID:          created.ID,
			DisplayName: "",
			Version:     created.Version,
		})
		require.NoError(t, err)
		assert.Equal(t, "user-1", updated.UserID)
		assert.Empty(t, updated.DisplayName)
		assert.Equal(t, domain.StatusActive, updated.Status)
		assert.Equal(t, created.Version+1, updated.Version)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.Empty(t, stored.DisplayName)
	})
}
