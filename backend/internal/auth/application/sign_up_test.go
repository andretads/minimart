package application

import (
	"minimart/internal/auth/domain"
	"minimart/internal/auth/infrastructure/memory"
	"minimart/pkg/metrics"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignUp(t *testing.T) {
	t.Run("Blank username is rejected", func(t *testing.T) {
		svc, _ := newTestService(t, nil)
		req := validSignUpRequest("")

		_, err := svc.SignUp(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUsername)
	})

	t.Run("Blank password is rejected", func(t *testing.T) {
		svc, _ := newTestService(t, nil)
		req := validSignUpRequest("john")
		req.Password = ""

		_, err := svc.SignUp(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyPassword)
	})

	t.Run("Password too short is rejected", func(t *testing.T) {
		svc, _ := newTestService(t, nil)
		req := validSignUpRequest("john")
		req.Password = "abc"

		_, err := svc.SignUp(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPasswordTooShort)
	})

	t.Run("Password too long is rejected", func(t *testing.T) {
		svc, _ := newTestService(t, nil)
		req := validSignUpRequest("john")
		req.Password = strings.Repeat("a", 73)

		_, err := svc.SignUp(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPasswordTooLong)
	})

	t.Run("Duplicate username is rejected", func(t *testing.T) {
		svc, _ := newTestService(t, nil)
		createTestAccount(t, svc, "john")
		req := validSignUpRequest("john")
		req.Email = "other@example.com"

		_, err := svc.SignUp(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAccountExists)
	})

	t.Run("Account created with generated ID and hashed password", func(t *testing.T) {
		svc, repo := newTestService(t, nil)
		seedStandardRoles(t, repo)

		acc, err := svc.SignUp(t.Context(), validSignUpRequest("john"))
		require.NoError(t, err)

		assert.NotEmpty(t, acc.ID)
		assert.Equal(t, "john", acc.Username)
		assert.True(t, acc.PasswordMatches("secret123"))
		assert.NotEmpty(t, acc.PasswordHash.Bytes())
		assert.False(t, acc.CreatedAt.IsZero())

		roles, err := repo.GetRolesForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Contains(t, roleIDs(roles), "customer")
	})

	t.Run("Customer provisioning failure does not fail signup", func(t *testing.T) {
		svc, _ := newTestService(t, &failingProvisioner{})
		logs := captureLogs(t)

		acc, err := svc.SignUp(t.Context(), validSignUpRequest("john"))
		require.NoError(t, err)
		assert.Equal(t, "john", acc.Username)
		assert.Contains(t, logs.String(), "level=WARN")
	})

	t.Run("Storage failure surfaces the error", func(t *testing.T) {
		svc := NewService(&failingSaveRepo{memory.New()}, nil, metrics.Noop{})

		_, err := svc.SignUp(t.Context(), validSignUpRequest("john"))
		require.Error(t, err)
		assert.EqualError(t, err, "disk on fire")
	})
}
