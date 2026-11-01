package application

import (
	"minimart/internal/cart/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateItemQuantity(t *testing.T) {
	t.Run("Blank user_id is rejected", func(t *testing.T) {
		env := newFeatureEnv(t)

		_, err := env.svc.UpdateItemQuantity(t.Context(), UpdateItemQuantityRequest{
			UserID:    "",
			ProductID: featureProductID,
			Quantity:  3,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIdRequired)
	})

	t.Run("Quantity less than 1 is rejected", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser1, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.UpdateItemQuantity(t.Context(), UpdateItemQuantityRequest{
			UserID:    featureUser1,
			ProductID: featureProductID,
			Quantity:  0,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuantity)
	})

	t.Run("Unknown product in cart returns item not found", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser1, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.UpdateItemQuantity(t.Context(), UpdateItemQuantityRequest{
			UserID:    featureUser1,
			ProductID: "nope",
			Quantity:  1,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCartItemNotFound)
	})

	t.Run("Updates quantity and bumps version", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser1, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.UpdateItemQuantity(t.Context(), UpdateItemQuantityRequest{
			UserID:    featureUser1,
			ProductID: featureProductID,
			Quantity:  5,
		})
		require.NoError(t, err)

		stored := env.storedCart(t, featureUser1)
		requireItem(t, stored.Items, featureProductID, featureProductName, 5, featureProductPriceCents)
		assert.Equal(t, 2, stored.Version)
	})

	t.Run("Update locks against the current cart version", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser1, 2, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.UpdateItemQuantity(t.Context(), UpdateItemQuantityRequest{
			UserID:    featureUser1,
			ProductID: featureProductID,
			Quantity:  3,
		})
		require.NoError(t, err)

		// The use case reads the current version (2) and CASes on it, so the
		// stored cart moves to version 3.
		stored := env.storedCart(t, featureUser1)
		requireItem(t, stored.Items, featureProductID, featureProductName, 3, featureProductPriceCents)
		assert.Equal(t, 3, stored.Version)
	})
}
