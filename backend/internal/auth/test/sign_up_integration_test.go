//go:build integration

package test

import (
	"errors"
	"minimart/internal/auth/domain"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignUpIntegration(t *testing.T) {
	t.Run("Blank username is rejected", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)

		_, err := svc.SignUp(t.Context(), validSignUpRequest(""))
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyUsername)
	})

	t.Run("Blank password is rejected", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)
		req := validSignUpRequest(uniqueUsername("john"))
		req.Password = ""

		_, err := svc.SignUp(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmptyPassword)
	})

	t.Run("Password too short is rejected", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)
		req := validSignUpRequest(uniqueUsername("john"))
		req.Password = "abc"

		_, err := svc.SignUp(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPasswordTooShort)
	})

	t.Run("Password too long is rejected", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)
		req := validSignUpRequest(uniqueUsername("john"))
		req.Password = strings.Repeat("a", 73)

		_, err := svc.SignUp(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPasswordTooLong)
	})

	t.Run("Duplicate username is rejected", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)
		username := uniqueUsername("john")
		newTestAccount(t, svc, username)

		req := validSignUpRequest(username)
		req.Email = uniqueUsername("other") + "@example.com"
		_, err := svc.SignUp(t.Context(), req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAccountExists)
	})

	t.Run("Account created with generated ID and hashed password", func(t *testing.T) {
		svc, repo := newAuthService(t, newRealCustomerProvisioner(t))
		username := uniqueUsername("john")

		acc, err := svc.SignUp(t.Context(), validSignUpRequest(username))
		require.NoError(t, err)

		assert.NotEmpty(t, acc.ID)
		assert.Equal(t, username, acc.Username)
		assert.True(t, acc.PasswordMatches("secret123"))
		assert.NotEmpty(t, acc.PasswordHash.Bytes())
		assert.False(t, acc.CreatedAt.IsZero())

		roles, err := repo.GetRolesForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Contains(t, roleIDs(roles), "customer")
	})

	t.Run("Customer provisioning failure does not fail signup", func(t *testing.T) {
		svc, _ := newAuthService(t, &fakeCustomerProvisioner{err: errors.New("customer service unavailable")})
		logs := captureLogs(t)
		req := validSignUpRequest(uniqueUsername("john"))

		acc, err := svc.SignUp(t.Context(), req)
		require.NoError(t, err)
		assert.Equal(t, req.Username, acc.Username)
		assert.Contains(t, logs.String(), "level=WARN")
	})

	t.Run("Storage failure surfaces the error", func(t *testing.T) {
		svc := newFailingAuthService(t)

		_, err := svc.SignUp(t.Context(), validSignUpRequest(uniqueUsername("john")))
		require.Error(t, err)
		assert.EqualError(t, err, "disk on fire")
	})
}
