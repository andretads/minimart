package application

import (
	"minimart/internal/auth/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAccount(t *testing.T) {
	t.Run("returns account by ID", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		acc, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})

		got, err := svc.GetAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Equal(t, acc.ID, got.ID)
	})

	t.Run("returns not found for missing ID", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		_, err := svc.GetAccount(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAccountNotFound)
	})
}
