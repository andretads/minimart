//go:build integration

package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReviewsWritePermissionSeeding proves the migration that introduces the
// reviews:write permission is applied: the permission exists, the customer and
// admin roles hold it, and a customer account can therefore submit a product
// review.
func TestReviewsWritePermissionSeeding(t *testing.T) {
	svc, repo := newAuthService(t, nil)

	perm, err := repo.GetPermission(t.Context(), "reviews:write")
	require.NoError(t, err)
	assert.Equal(t, "reviews:write", perm.ID)

	t.Run("customer and admin roles hold it", func(t *testing.T) {
		readPool, _ := mustPools(t)
		rows, err := readPool.Query(t.Context(), `
			SELECT role_id FROM role_permission WHERE permission_id = 'reviews:write' ORDER BY role_id
		`)
		require.NoError(t, err)
		defer rows.Close()

		var roles []string
		for rows.Next() {
			var roleID string
			require.NoError(t, rows.Scan(&roleID))
			roles = append(roles, roleID)
		}
		require.NoError(t, rows.Err())
		assert.Equal(t, []string{"admin", "customer"}, roles)
	})

	t.Run("customer role can submit reviews", func(t *testing.T) {
		acc := newTestAccount(t, svc, uniqueUsername("rev-it"))
		require.NoError(t, svc.AssignDefaultRole(t.Context(), acc.ID))

		perms, err := svc.PermissionsForAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Contains(t, perms, "reviews:write")
		assert.Contains(t, perms, "catalog:read")
	})
}
