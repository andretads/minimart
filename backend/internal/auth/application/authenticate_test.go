package application

import (
	"minimart/internal/auth/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticate(t *testing.T) {
	t.Run("authenticates valid credentials", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		acc, _ := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})

		got, err := svc.Authenticate(t.Context(), AuthenticateRequest{Username: "testuser", Password: "password123"})

		require.NoError(t, err)
		assert.Equal(t, acc.ID, got.ID)
	})

	t.Run("rejects wrong password", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		_, _ = svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})

		_, err := svc.Authenticate(t.Context(), AuthenticateRequest{Username: "testuser", Password: "wrong"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("rejects unknown username without enumeration", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		_, err := svc.Authenticate(t.Context(), AuthenticateRequest{Username: "nobody", Password: "password123"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})
}
