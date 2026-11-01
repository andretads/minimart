//go:build integration

package test

import (
	"crypto/rand"
	"minimart/internal/auth/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionsForAccountIntegration(t *testing.T) {
	t.Run("Account with roles returns seeded permission IDs", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)
		acc := newTestAccount(t, svc, uniqueUsername("acc"))
		require.NoError(t, svc.AssignRole(t.Context(), acc.ID, "support"))

		perms, err := svc.PermissionsForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Subset(t, perms, []string{"catalog:read", "orders:read", "customers:read"})
	})

	t.Run("Customer role includes customer read", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)
		acc := newTestAccount(t, svc, uniqueUsername("acc-customer"))

		perms, err := svc.PermissionsForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Subset(t, perms, []string{"customers:read", "customers:write"})
	})

	t.Run("Unknown account returns empty list", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)

		perms, err := svc.PermissionsForAccount(t.Context(), "unknown")
		require.NoError(t, err)
		assert.Empty(t, perms)
	})

	t.Run("Role-less account returns empty list", func(t *testing.T) {
		svc, repo := newAuthService(t, nil)
		username := uniqueUsername("acc-roleless")
		acc, err := domain.NewAccount(rand.Text(), username, "secret123", username+"@example.com")
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), acc))

		perms, err := svc.PermissionsForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Empty(t, perms)
	})
}
