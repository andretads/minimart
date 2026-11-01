//go:build integration

package test

import (
	"minimart/internal/auth/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignRoleIntegration(t *testing.T) {
	t.Run("Role assigned to account", func(t *testing.T) {
		svc, repo := newAuthService(t, nil)
		acc := newTestAccount(t, svc, uniqueUsername("acc"))

		err := svc.AssignRole(t.Context(), acc.ID, "support")
		require.NoError(t, err)

		roles, err := repo.GetRolesForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Contains(t, roleIDs(roles), "support")
	})

	t.Run("Assigning twice is idempotent", func(t *testing.T) {
		svc, repo := newAuthService(t, nil)
		acc := newTestAccount(t, svc, uniqueUsername("acc"))

		require.NoError(t, svc.AssignRole(t.Context(), acc.ID, "support"))
		require.NoError(t, svc.AssignRole(t.Context(), acc.ID, "support"))

		roles, err := repo.GetRolesForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, countRole(roles, "support"))
	})

	t.Run("Unknown role returns role not found", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)
		acc := newTestAccount(t, svc, uniqueUsername("acc"))

		err := svc.AssignRole(t.Context(), acc.ID, "unknown-role")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRoleNotFound)
	})
}
