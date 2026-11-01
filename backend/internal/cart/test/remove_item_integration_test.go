//go:build integration

package test

import (
	"minimart/internal/cart/application"
	"minimart/internal/cart/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveItemIntegration(t *testing.T) {
	t.Run("Blank user_id is rejected", func(t *testing.T) {
		env := newCartEnv(t)

		_, err := env.svc.RemoveItem(t.Context(), application.RemoveItemRequest{
			UserID:    "",
			ProductID: featureProductID,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrUserIdRequired)
	})

	t.Run("Unknown product in cart returns item not found", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()
		env.seedCart(t, user, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.RemoveItem(t.Context(), application.RemoveItemRequest{
			UserID:    user,
			ProductID: "nope",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCartItemNotFound)
	})

	t.Run("Removes item from cart", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()
		env.seedCart(t, user, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.RemoveItem(t.Context(), application.RemoveItemRequest{
			UserID:    user,
			ProductID: featureProductID,
		})
		require.NoError(t, err)

		stored := env.storedCart(t, user)
		assert.Empty(t, stored.Items)
		assert.Equal(t, domain.StatusActive, stored.Status)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Removing last item leaves an empty active cart", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()
		env.seedCart(t, user, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 1))

		_, err := env.svc.RemoveItem(t.Context(), application.RemoveItemRequest{
			UserID:    user,
			ProductID: featureProductID,
		})
		require.NoError(t, err)

		stored := env.storedCart(t, user)
		assert.Empty(t, stored.Items)
		assert.Equal(t, domain.StatusActive, stored.Status)
	})
}
