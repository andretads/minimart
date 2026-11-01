package application

import (
	"minimart/internal/cart/domain"
	"testing"

	catalogclient "minimart/internal/catalog/presentation/module/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddItem(t *testing.T) {
	t.Run("Blank user_id is rejected", func(t *testing.T) {
		env := newFeatureEnv(t)

		_, err := env.svc.AddItem(t.Context(), AddItemRequest{
			UserID:    "",
			ProductID: featureProductID,
			Quantity:  1,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIdRequired)
	})

	t.Run("Non-positive quantity is rejected", func(t *testing.T) {
		env := newFeatureEnv(t)

		_, err := env.svc.AddItem(t.Context(), AddItemRequest{
			UserID:    featureUser1,
			ProductID: featureProductID,
			Quantity:  0,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuantity)
	})

	t.Run("Unknown product returns catalog error", func(t *testing.T) {
		env := newFeatureEnv(t)

		_, err := env.svc.AddItem(t.Context(), AddItemRequest{
			UserID:    featureUser1,
			ProductID: "nope",
			Quantity:  1,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, catalogclient.ErrProductNotFound)
	})

	t.Run("Creates cart with catalog price/name snapshot on first add", func(t *testing.T) {
		env := newFeatureEnv(t)

		_, err := env.svc.AddItem(t.Context(), AddItemRequest{
			UserID:    featureUser1,
			ProductID: featureProductID,
			Quantity:  2,
		})
		require.NoError(t, err)

		stored := env.storedCart(t, featureUser1)
		requireItem(t, stored.Items, featureProductID, featureProductName, 2, featureProductPriceCents)
		assert.Equal(t, domain.StatusActive, stored.Status)
	})

	t.Run("Second add of same product increments quantity", func(t *testing.T) {
		env := newFeatureEnv(t)

		_, err := env.svc.AddItem(t.Context(), AddItemRequest{
			UserID:    featureUser1,
			ProductID: featureProductID,
			Quantity:  1,
		})
		require.NoError(t, err)

		_, err = env.svc.AddItem(t.Context(), AddItemRequest{
			UserID:    featureUser1,
			ProductID: featureProductID,
			Quantity:  3,
		})
		require.NoError(t, err)

		stored := env.storedCart(t, featureUser1)
		requireItem(t, stored.Items, featureProductID, featureProductName, 4, featureProductPriceCents)
	})

	t.Run("Add to consumed cart starts a new active cart", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.consumedCart(t, featureUser1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.AddItem(t.Context(), AddItemRequest{
			UserID:    featureUser1,
			ProductID: featureProductID,
			Quantity:  1,
		})
		require.NoError(t, err)

		// The consumed cart is not reused: adding after checkout starts the
		// customer's next active cart.
		active := env.activeCart(t, featureUser1)
		requireItem(t, active.Items, featureProductID, featureProductName, 1, featureProductPriceCents)
		assert.Equal(t, domain.StatusActive, active.Status)
	})
}
