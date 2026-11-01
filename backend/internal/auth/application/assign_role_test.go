package application

import (
	"minimart/internal/auth/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignRole(t *testing.T) {
	t.Run("Role assigned to account", func(t *testing.T) {
		svc, repo := newTestService(t, nil)
		seedStandardRoles(t, repo)
		acc := createTestAccount(t, svc, "acc-1")

		err := svc.AssignRole(t.Context(), acc.ID, "support")
		require.NoError(t, err)

		roles, err := repo.GetRolesForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Contains(t, roleIDs(roles), "support")
	})

	t.Run("Assigning twice is idempotent", func(t *testing.T) {
		svc, repo := newTestService(t, nil)
		seedStandardRoles(t, repo)
		acc := createTestAccount(t, svc, "acc-1")

		require.NoError(t, svc.AssignRole(t.Context(), acc.ID, "support"))
		require.NoError(t, svc.AssignRole(t.Context(), acc.ID, "support"))

		roles, err := repo.GetRolesForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, countRole(roles, "support"))
	})

	t.Run("Unknown role returns role not found", func(t *testing.T) {
		svc, repo := newTestService(t, nil)
		seedStandardRoles(t, repo)
		acc := createTestAccount(t, svc, "acc-1")

		err := svc.AssignRole(t.Context(), acc.ID, "unknown-role")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRoleNotFound)
	})
}
