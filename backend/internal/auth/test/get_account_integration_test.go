//go:build integration

package test

import (
	"minimart/internal/auth/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAccountIntegration(t *testing.T) {
	t.Run("Unknown ID returns account not found", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)

		_, err := svc.GetAccount(t.Context(), "missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAccountNotFound)
	})

	t.Run("Existing account returned", func(t *testing.T) {
		svc, _ := newAuthService(t, nil)
		username := uniqueUsername("john")
		acc := newTestAccount(t, svc, username)

		got, err := svc.GetAccount(t.Context(), acc.ID)
		require.NoError(t, err)
		assert.Equal(t, acc.ID, got.ID)
		assert.Equal(t, username, got.Username)
	})
}
