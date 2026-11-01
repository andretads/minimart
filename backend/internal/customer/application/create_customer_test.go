package application

import (
	"minimart/internal/customer/domain"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCustomer(t *testing.T) {
	t.Run("Blank user_id is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateCustomerRequest()
		req.UserID = ""

		_, err := svc.CreateCustomer(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUserID)
	})

	t.Run("Display name too long is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateCustomerRequest()
		req.DisplayName = strings.Repeat("a", 201)

		_, err := svc.CreateCustomer(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrDisplayNameTooLong)
	})

	t.Run("Customer got created with no problems", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateCustomerRequest()
		created := createTestCustomer(t, svc, req)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, "user-1", stored.UserID)
		assert.Equal(t, "John", stored.DisplayName)
		assert.Equal(t, domain.StatusActive, stored.Status)
		assert.False(t, stored.CreatedAt.IsZero())
	})

	t.Run("Customer without display name gets created", func(t *testing.T) {
		svc, repo := newTestService(t)

		req := validCreateCustomerRequest()
		req.UserID = "user-2"
		req.DisplayName = ""
		created := createTestCustomer(t, svc, req)

		stored, err := repo.GetByID(t.Context(), created.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, stored.ID)
		assert.Equal(t, "user-2", stored.UserID)
		assert.Empty(t, stored.DisplayName)
		assert.Equal(t, domain.StatusActive, stored.Status)
		assert.False(t, stored.CreatedAt.IsZero())
	})

	t.Run("Duplicate user_id is rejected", func(t *testing.T) {
		svc, _ := newTestService(t)

		req := validCreateCustomerRequest()
		_, err := svc.CreateCustomer(t.Context(), req)
		require.NoError(t, err)

		req2 := validCreateCustomerRequest()
		req2.DisplayName = "Jane"
		_, err = svc.CreateCustomer(t.Context(), req2)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCustomerExists)
	})
}
