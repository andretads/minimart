package application

import (
	"minimart/internal/auth/domain"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticate(t *testing.T) {
	t.Run("Unknown username returns invalid credentials", func(t *testing.T) {
		svc, _ := newTestService(t, nil)

		_, err := svc.Authenticate(t.Context(), AuthenticateRequest{Username: "unknown", Password: "secret123"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("Wrong password returns invalid credentials", func(t *testing.T) {
		svc, _ := newTestService(t, nil)
		createTestAccount(t, svc, "john")

		_, err := svc.Authenticate(t.Context(), AuthenticateRequest{Username: "john", Password: "wrong"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("Valid credentials return the account", func(t *testing.T) {
		svc, _ := newTestService(t, nil)
		acc := createTestAccount(t, svc, "john")

		got, err := svc.Authenticate(t.Context(), AuthenticateRequest{Username: "john", Password: "secret123"})
		require.NoError(t, err)
		assert.Equal(t, acc.ID, got.ID)
		assert.Equal(t, "john", got.Username)
	})

	t.Run("Login best-effort ensures customer", func(t *testing.T) {
		svc, repo := newTestService(t, nil)
		acc := createTestAccount(t, svc, "john")
		logs := captureLogs(t)

		failing := NewService(repo, &failingProvisioner{}, metrics.Noop{})
		got, err := failing.Authenticate(t.Context(), AuthenticateRequest{Username: "john", Password: "secret123"})
		require.NoError(t, err)
		assert.Equal(t, acc.ID, got.ID)
		assert.Equal(t, "john", got.Username)
		assert.Contains(t, logs.String(), "level=WARN")
	})
}
