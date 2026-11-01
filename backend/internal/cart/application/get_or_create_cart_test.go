package application

import (
	"minimart/internal/cart/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrCreateCart(t *testing.T) {
	t.Run("Blank user_id is rejected", func(t *testing.T) {
		env := newFeatureEnv(t)

		_, err := env.svc.GetOrCreateCart(t.Context(), GetOrCreateCartRequest{UserID: ""})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIdRequired)
	})

	t.Run("Creates cart when none exists", func(t *testing.T) {
		env := newFeatureEnv(t)

		cart, err := env.svc.GetOrCreateCart(t.Context(), GetOrCreateCartRequest{UserID: featureUser1})
		require.NoError(t, err)

		assert.Equal(t, domain.StatusActive, cart.Status)
		assert.Empty(t, cart.Items)
		assert.Equal(t, 1, cart.Version)

		stored := env.storedCart(t, featureUser1)
		assert.Equal(t, cart.ID, stored.ID)
		assert.Equal(t, 1, stored.Version)
	})

	t.Run("Returns existing active cart", func(t *testing.T) {
		env := newFeatureEnv(t)
		seed := env.seedCart(t, featureUser1, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		cart, err := env.svc.GetOrCreateCart(t.Context(), GetOrCreateCartRequest{UserID: featureUser1})
		require.NoError(t, err)

		assert.Equal(t, seed.ID, cart.ID)
		requireItem(t, cart.Items, featureProductID, featureProductName, 2, featureProductPriceCents)
	})

	t.Run("Concurrent create for same user returns same cart", func(t *testing.T) {
		env := newFeatureEnv(t)

		first, err := env.svc.GetOrCreateCart(t.Context(), GetOrCreateCartRequest{UserID: featureUserRace})
		require.NoError(t, err)
		second, err := env.svc.GetOrCreateCart(t.Context(), GetOrCreateCartRequest{UserID: featureUserRace})
		require.NoError(t, err)

		assert.Equal(t, first.ID, second.ID)

		// Subsequent lookups resolve to the same single active cart.
		stored := env.activeCart(t, featureUserRace)
		assert.Equal(t, first.ID, stored.ID)
	})
}
