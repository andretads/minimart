package application

import (
	"minimart/internal/customer/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureCustomer(t *testing.T) {
	t.Run("Empty user_id is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.EnsureCustomer(t.Context(), "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Existing customer returned unchanged", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateCustomerRequest()
		req.UserID = "user-1"
		req.DisplayName = "John"
		created := createTestCustomer(t, svc, req)

		got, err := svc.EnsureCustomer(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "user-1", got.UserID)
		assert.Equal(t, "John", got.DisplayName)
		assert.Equal(t, domain.StatusActive, got.Status)

		stored, err := repo.GetByUserID(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, created.ID, stored.ID)
	})

	t.Run("Missing customer created with active status and empty display name", func(t *testing.T) {
		svc, repo := newTestService(t)

		got, err := svc.EnsureCustomer(t.Context(), "user-new")
		require.NoError(t, err)
		assert.NotEmpty(t, got.ID)
		assert.Equal(t, "user-new", got.UserID)
		assert.Empty(t, got.DisplayName)
		assert.Equal(t, domain.StatusActive, got.Status)
		assert.Equal(t, 1, got.Version)

		stored, err := repo.GetByUserID(t.Context(), "user-new")
		require.NoError(t, err)
		assert.Equal(t, got.ID, stored.ID)
	})

	t.Run("Concurrent create for same user_id returns the same customer", func(t *testing.T) {
		svc, repo := newTestService(t)

		c1, err := svc.EnsureCustomer(t.Context(), "user-race")
		require.NoError(t, err)
		c2, err := svc.EnsureCustomer(t.Context(), "user-race")
		require.NoError(t, err)

		assert.Equal(t, c1.ID, c2.ID)

		stored, err := repo.GetByUserID(t.Context(), "user-race")
		require.NoError(t, err)
		assert.Equal(t, c1.ID, stored.ID)
	})
}
