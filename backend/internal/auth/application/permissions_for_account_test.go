package application

import (
	"minimart/internal/auth/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionsForAccount(t *testing.T) {
	t.Run("Account with roles returns seeded permission IDs", func(t *testing.T) {
		svc, repo := newTestService(t, nil)
		seedStandardRoles(t, repo)
		seedPermissions(t, repo, "support", "catalog:read", "orders:read", "customers:read")
		acc := createTestAccount(t, svc, "acc-1")
		require.NoError(t, svc.AssignRole(t.Context(), acc.ID, "support"))

		perms, err := svc.PermissionsForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Subset(t, perms, []string{"catalog:read", "orders:read", "customers:read"})
	})

	t.Run("Customer role includes customer read", func(t *testing.T) {
		svc, repo := newTestService(t, nil)
		seedStandardRoles(t, repo)
		seedPermissions(t, repo, "customer", "customers:read", "customers:write")
		acc := createTestAccount(t, svc, "acc-customer")

		perms, err := svc.PermissionsForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Subset(t, perms, []string{"customers:read", "customers:write"})
	})

	t.Run("Unknown account returns empty list", func(t *testing.T) {
		svc, _ := newTestService(t, nil)

		perms, err := svc.PermissionsForAccount(t.Context(), "unknown")
		require.NoError(t, err)
		assert.Empty(t, perms)
	})

	t.Run("Role-less account returns empty list", func(t *testing.T) {
		svc, repo := newTestService(t, nil)
		seedStandardRoles(t, repo)

		acc, err := domain.NewAccount(newID(), "acc-1", "secret123", "acc-1@example.com")
		require.NoError(t, err)
		require.NoError(t, repo.Save(t.Context(), acc))

		perms, err := svc.PermissionsForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Empty(t, perms)
	})
}
