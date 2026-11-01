//go:build integration

package test

import (
	"errors"
	"minimart/internal/auth/application"
	"minimart/internal/auth/domain"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticateIntegration(t *testing.T) {
	t.Run("Unknown username returns invalid credentials", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)

		_, err := svc.Authenticate(t.Context(), application.AuthenticateRequest{
			Username: uniqueUsername("unknown"),
			Password: "secret123",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("Wrong password returns invalid credentials", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)
		username := uniqueUsername("john")
		newTestAccount(t, svc, username)

		_, err := svc.Authenticate(t.Context(), application.AuthenticateRequest{
			Username: username,
			Password: "wrong",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("Valid credentials return the account", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)
		username := uniqueUsername("john")
		acc := newTestAccount(t, svc, username)

		got, err := svc.Authenticate(t.Context(), application.AuthenticateRequest{
			Username: username,
			Password: "secret123",
		})
		require.NoError(t, err)
		assert.Equal(t, acc.ID, got.ID)
		assert.Equal(t, username, got.Username)
	})

	t.Run("Login best-effort ensures customer", func(t *testing.T) {
		svc, repo := newAuthService(t, nil)
		username := uniqueUsername("john")
		acc := newTestAccount(t, svc, username)
		logs := captureLogs(t)

		failing := application.NewService(repo, &fakeCustomerProvisioner{err: errors.New("customer service unavailable")}, metrics.Noop{})
		got, err := failing.Authenticate(t.Context(), application.AuthenticateRequest{
			Username: username,
			Password: "secret123",
		})
		require.NoError(t, err)
		assert.Equal(t, acc.ID, got.ID)
		assert.Equal(t, username, got.Username)
		assert.Contains(t, logs.String(), "level=WARN")
	})
}
