package application

import (
	"minimart/internal/auth/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureAdmins(t *testing.T) {
	t.Run("Empty list is a no-op", func(t *testing.T) {
		svc, repo := newTestService(t, nil)
		seedStandardRoles(t, repo)

		err := svc.EnsureAdmins(t.Context(), []string{})
		require.NoError(t, err)

		roles, err := repo.GetRolesForAccount(t.Context(), "acc-1")
		require.NoError(t, err)
		assert.Empty(t, roles)
	})

	t.Run("Listed usernames get admin role", func(t *testing.T) {
		svc, repo := newTestService(t, nil)
		seedStandardRoles(t, repo)
		alice := createTestAccount(t, svc, "alice")
		bob := createTestAccount(t, svc, "bob")
		carol := createTestAccount(t, svc, "carol")

		err := svc.EnsureAdmins(t.Context(), []string{"alice", "bob"})
		require.NoError(t, err)

		aliceRoles, err := repo.GetRolesForAccount(t.Context(), alice.ID)
		require.NoError(t, err)
		assert.Contains(t, roleIDs(aliceRoles), "admin")

		bobRoles, err := repo.GetRolesForAccount(t.Context(), bob.ID)
		require.NoError(t, err)
		assert.Contains(t, roleIDs(bobRoles), "admin")

		carolRoles, err := repo.GetRolesForAccount(t.Context(), carol.ID)
		require.NoError(t, err)
		assert.NotContains(t, roleIDs(carolRoles), "admin")
	})

	t.Run("Already-admin is tolerated", func(t *testing.T) {
		svc, repo := newTestService(t, nil)
		seedStandardRoles(t, repo)
		alice := createTestAccount(t, svc, "alice")
		require.NoError(t, svc.AssignRole(t.Context(), alice.ID, "admin"))

		err := svc.EnsureAdmins(t.Context(), []string{"alice"})
		require.NoError(t, err)

		roles, err := repo.GetRolesForAccount(t.Context(), alice.ID)
		require.NoError(t, err)
		assert.Contains(t, roleIDs(roles), "admin")
	})

	t.Run("Unknown usernames are skipped", func(t *testing.T) {
		svc, repo := newTestService(t, nil)
		seedStandardRoles(t, repo)
		alice := createTestAccount(t, svc, "alice")

		err := svc.EnsureAdmins(t.Context(), []string{"alice", "unknown-user"})
		require.NoError(t, err)

		roles, err := repo.GetRolesForAccount(t.Context(), alice.ID)
		require.NoError(t, err)
		assert.Contains(t, roleIDs(roles), "admin")

		_, err = repo.GetByUsername(t.Context(), "unknown-user")
		assert.ErrorIs(t, err, domain.ErrAccountNotFound)
	})
}
