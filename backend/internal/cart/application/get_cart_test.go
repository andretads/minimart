package application

import (
	"minimart/internal/cart/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCart(t *testing.T) {
	t.Run("Blank user_id is rejected", func(t *testing.T) {
		env := newFeatureEnv(t)

		_, err := env.svc.GetCart(t.Context(), GetCartRequest{UserID: ""})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIdRequired)
	})

	t.Run("Customer has no cart returns cart not found", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.ensureCustomer(t, featureUserNoCart)

		_, err := env.svc.GetCart(t.Context(), GetCartRequest{UserID: featureUserNoCart})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCartNotFound)
	})

	t.Run("Returns cart with items", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser1, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		cart, err := env.svc.GetCart(t.Context(), GetCartRequest{UserID: featureUser1})
		require.NoError(t, err)

		requireItem(t, cart.Items, featureProductID, featureProductName, 2, featureProductPriceCents)
	})
}
