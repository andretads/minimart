//go:build integration

package test

import (
	"minimart/internal/cart/application"
	"minimart/internal/cart/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClearCartIntegration(t *testing.T) {
	t.Run("Blank user_id is rejected", func(t *testing.T) {
		env := newCartEnv(t)

		_, err := env.svc.ClearCart(t.Context(), application.ClearCartRequest{UserID: ""})

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrUserIdRequired)
	})

	t.Run("Already-empty cart is a no-op", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()
		_, err := env.svc.GetOrCreateCart(t.Context(), application.GetOrCreateCartRequest{UserID: user})
		require.NoError(t, err)

		_, err = env.svc.ClearCart(t.Context(), application.ClearCartRequest{UserID: user})
		require.NoError(t, err)

		// Clearing an empty cart succeeds and still persists a new version.
		stored := env.storedCart(t, user)
		assert.Empty(t, stored.Items)
		assert.Equal(t, domain.StatusActive, stored.Status)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Removes all items, cart stays active", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()
		env.seedCart(t, user, 1,
			featureItem(featureProductID, featureProductName, featureProductPriceCents, 2),
			featureItem(featureExtraProductID, featureExtraProductName, featureExtraProductPrice, 1),
		)

		_, err := env.svc.ClearCart(t.Context(), application.ClearCartRequest{UserID: user})
		require.NoError(t, err)

		stored := env.storedCart(t, user)
		assert.Empty(t, stored.Items)
		assert.Equal(t, domain.StatusActive, stored.Status)
		assert.Equal(t, 2, stored.Version)
	})
}
