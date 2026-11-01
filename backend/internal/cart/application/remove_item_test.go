package application

import (
	"minimart/internal/cart/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveItem(t *testing.T) {
	t.Run("Blank user_id is rejected", func(t *testing.T) {
		env := newFeatureEnv(t)

		_, err := env.svc.RemoveItem(t.Context(), RemoveItemRequest{
			UserID:    "",
			ProductID: featureProductID,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIdRequired)
	})

	t.Run("Unknown product in cart returns item not found", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser1, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.RemoveItem(t.Context(), RemoveItemRequest{
			UserID:    featureUser1,
			ProductID: "nope",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCartItemNotFound)
	})

	t.Run("Removes item from cart", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser1, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.RemoveItem(t.Context(), RemoveItemRequest{
			UserID:    featureUser1,
			ProductID: featureProductID,
		})
		require.NoError(t, err)

		stored := env.storedCart(t, featureUser1)
		assert.Empty(t, stored.Items)
		assert.Equal(t, domain.StatusActive, stored.Status)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Removing last item leaves an empty active cart", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser2, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 1))

		_, err := env.svc.RemoveItem(t.Context(), RemoveItemRequest{
			UserID:    featureUser2,
			ProductID: featureProductID,
		})
		require.NoError(t, err)

		stored := env.storedCart(t, featureUser2)
		assert.Empty(t, stored.Items)
		assert.Equal(t, domain.StatusActive, stored.Status)
	})
}
