package application

import (
	"minimart/internal/auth/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignUp(t *testing.T) {
	t.Run("creates an account and provisions a customer", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		acc, err := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})
		require.NoError(t, err)
		assert.NotEmpty(t, acc.ID)
		assert.Equal(t, "testuser", acc.Username)
		assert.Equal(t, "test@example.com", acc.Email)
		assert.True(t, acc.PasswordMatches("password123"))
	})

	t.Run("rejects duplicate username", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		_, _ = svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})
		_, err := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "different", Email: "test2@example.com"})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAccountExists)
	})

	t.Run("rejects duplicate email", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		_, _ = svc.SignUp(t.Context(), SignUpRequest{Username: "testuser1", Password: "password123", Email: "test@example.com"})
		_, err := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser2", Password: "different", Email: "test@example.com"})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmailExists)
	})

	t.Run("rejects empty username", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		_, err := svc.SignUp(t.Context(), SignUpRequest{Username: "", Password: "password123", Email: "test@example.com"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUsername)
	})

	t.Run("rejects short password", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		_, err := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "12345", Email: "test@example.com"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPasswordTooShort)
	})

	t.Run("rejects empty email", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		_, err := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: ""})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyEmail)
	})

	t.Run("rejects invalid email", func(t *testing.T) {
		svc, _ := newTestService(t, &fakeProvisioner{})

		_, err := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "invalid-email"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidEmail)
	})

	t.Run("continues on customer provision failure", func(t *testing.T) {
		svc, _ := newTestService(t, &failingProvisioner{})

		acc, err := svc.SignUp(t.Context(), SignUpRequest{Username: "testuser", Password: "password123", Email: "test@example.com"})
		require.NoError(t, err)
		assert.Equal(t, "testuser", acc.Username)
		assert.Equal(t, "test@example.com", acc.Email)
	})
}
