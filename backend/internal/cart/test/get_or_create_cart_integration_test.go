//go:build integration

package test

import (
	"minimart/internal/cart/application"
	"minimart/internal/cart/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrCreateCartIntegration(t *testing.T) {
	t.Run("Blank user_id is rejected", func(t *testing.T) {
		env := newCartEnv(t)

		_, err := env.svc.GetOrCreateCart(t.Context(), application.GetOrCreateCartRequest{UserID: ""})

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrUserIdRequired)
	})

	t.Run("Creates cart when none exists", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()

		cart, err := env.svc.GetOrCreateCart(t.Context(), application.GetOrCreateCartRequest{UserID: user})
		require.NoError(t, err)

		assert.Equal(t, domain.StatusActive, cart.Status)
		assert.Empty(t, cart.Items)
		assert.Equal(t, 1, cart.Version)

		stored := env.storedCart(t, user)
		assert.Equal(t, cart.ID, stored.ID)
		assert.Equal(t, "cust-"+user, stored.CustomerID)
		assert.Equal(t, domain.StatusActive, stored.Status)
		assert.Empty(t, stored.Items)
		assert.Equal(t, 1, stored.Version)
	})

	t.Run("Returns existing active cart", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()
		seed := env.seedCart(t, user, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		cart, err := env.svc.GetOrCreateCart(t.Context(), application.GetOrCreateCartRequest{UserID: user})
		require.NoError(t, err)

		assert.Equal(t, seed.ID, cart.ID)
		requireItem(t, cart.Items, featureProductID, featureProductName, 2, featureProductPriceCents)
	})

	t.Run("Concurrent create for same user returns same cart", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()

		first, err := env.svc.GetOrCreateCart(t.Context(), application.GetOrCreateCartRequest{UserID: user})
		require.NoError(t, err)
		second, err := env.svc.GetOrCreateCart(t.Context(), application.GetOrCreateCartRequest{UserID: user})
		require.NoError(t, err)

		assert.Equal(t, first.ID, second.ID)

		// The unique partial index proves exactly one active cart exists: a
		// second active cart for the same customer is rejected.
		duplicate, err := domain.NewCart("cart-duplicate-"+user, "cust-"+user)
		require.NoError(t, err)
		assert.ErrorIs(t, env.repo.Create(t.Context(), duplicate), domain.ErrCartAlreadyExists)
	})
}
