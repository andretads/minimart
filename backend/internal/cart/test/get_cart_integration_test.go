//go:build integration

package test

import (
	"minimart/internal/cart/application"
	"minimart/internal/cart/domain"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCartIntegration(t *testing.T) {
	t.Run("Blank user_id is rejected", func(t *testing.T) {
		env := newCartEnv(t)

		_, err := env.svc.GetCart(t.Context(), application.GetCartRequest{UserID: ""})

		require.Error(t, err)
		assert.ErrorIs(t, err, application.ErrUserIdRequired)
	})

	t.Run("Customer has no cart returns cart not found", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()
		env.ensureCustomer(t, user)

		_, err := env.svc.GetCart(t.Context(), application.GetCartRequest{UserID: user})

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCartNotFound)
	})

	t.Run("Returns cart with items", func(t *testing.T) {
		env := newCartEnv(t)
		user := uniqueUser()
		env.seedCart(t, user, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		cart, err := env.svc.GetCart(t.Context(), application.GetCartRequest{UserID: user})
		require.NoError(t, err)

		requireItem(t, cart.Items, featureProductID, featureProductName, 2, featureProductPriceCents)
	})
}
