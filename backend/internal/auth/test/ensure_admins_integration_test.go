//go:build integration

package test

import (
	"minimart/internal/auth/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureAdminsIntegration(t *testing.T) {
	t.Run("Empty list is a no-op", func(t *testing.T) {
		svc, repo := newAuthService(t, nil)

		err := svc.EnsureAdmins(t.Context(), []string{})
		require.NoError(t, err)

		_, err = repo.GetByUsername(t.Context(), uniqueUsername("admin"))
		assert.ErrorIs(t, err, domain.ErrAccountNotFound)
	})

	t.Run("Listed usernames get admin role", func(t *testing.T) {
		svc, repo := newAuthService(t, nil)
		aliceName := uniqueUsername("alice")
		bobName := uniqueUsername("bob")
		carolName := uniqueUsername("carol")
		alice := newTestAccount(t, svc, aliceName)
		bob := newTestAccount(t, svc, bobName)
		carol := newTestAccount(t, svc, carolName)

		err := svc.EnsureAdmins(t.Context(), []string{aliceName, bobName})
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
		svc, repo := newAuthService(t, nil)
		username := uniqueUsername("alice")
		alice := newTestAccount(t, svc, username)
		require.NoError(t, svc.AssignRole(t.Context(), alice.ID, "admin"))

		err := svc.EnsureAdmins(t.Context(), []string{username})
		require.NoError(t, err)

		roles, err := repo.GetRolesForAccount(t.Context(), alice.ID)
		require.NoError(t, err)
		assert.Contains(t, roleIDs(roles), "admin")
	})

	t.Run("Unknown usernames are skipped", func(t *testing.T) {
		svc, repo := newAuthService(t, nil)
		aliceName := uniqueUsername("alice")
		unknownName := uniqueUsername("unknown-user")
		alice := newTestAccount(t, svc, aliceName)

		err := svc.EnsureAdmins(t.Context(), []string{aliceName, unknownName})
		require.NoError(t, err)

		roles, err := repo.GetRolesForAccount(t.Context(), alice.ID)
		require.NoError(t, err)
		assert.Contains(t, roleIDs(roles), "admin")

		_, err = repo.GetByUsername(t.Context(), unknownName)
		assert.ErrorIs(t, err, domain.ErrAccountNotFound)
	})
}
