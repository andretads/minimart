package application

import (
	"minimart/internal/cart/domain"
	"minimart/pkg/metrics"
	"testing"

	orderclient "minimart/internal/order/presentation/module/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckoutCart(t *testing.T) {
	t.Run("Blank user_id is rejected", func(t *testing.T) {
		env := newFeatureEnv(t)

		_, err := env.svc.CheckoutCart(t.Context(), featureCheckoutRequest("", "k1"))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserIdRequired)
	})

	t.Run("Missing Idempotency-Key is rejected", func(t *testing.T) {
		env := newFeatureEnv(t)

		_, err := env.svc.CheckoutCart(t.Context(), featureCheckoutRequest(featureUser1, ""))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrIdempotencyKeyRequired)
	})

	t.Run("Empty cart returns cart empty", func(t *testing.T) {
		env := newFeatureEnv(t)
		_, err := env.svc.GetOrCreateCart(t.Context(), GetOrCreateCartRequest{UserID: featureUser2})
		require.NoError(t, err)

		_, err = env.svc.CheckoutCart(t.Context(), featureCheckoutRequest(featureUser2, "k1"))

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCartEmpty)
	})

	t.Run("Places order via contract, cart becomes consumed with order id", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser1, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		ord, err := env.svc.CheckoutCart(t.Context(), featureCheckoutRequest(featureUser1, "k1"))
		require.NoError(t, err)

		require.Len(t, env.order.lastRequest.Items, 1)
		assert.Equal(t, featureProductID, env.order.lastRequest.Items[0].ProductID)
		assert.Equal(t, 2, env.order.lastRequest.Items[0].Quantity)
		assert.Equal(t, int64(1000), ord.TotalCents)

		stored := env.storedCart(t, featureUser1)
		assert.Equal(t, domain.StatusConsumed, stored.Status)
		assert.Equal(t, ord.ID, stored.OrderID)
	})

	t.Run("The shipping is priced against the cart subtotal", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser1, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		ord, err := env.svc.CheckoutCart(t.Context(), featureCheckoutRequest(featureUser1, "k1"))
		require.NoError(t, err)

		assert.Equal(t, int64(1000), ord.TotalCents)
		assert.Equal(t, int64(599), ord.ShippingCostCents)
		assert.Equal(t, int64(599), env.order.lastRequest.ShippingCostCents)
	})

	t.Run("A cart at the free shipping threshold ships free", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser1, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 12))

		ord, err := env.svc.CheckoutCart(t.Context(), featureCheckoutRequest(featureUser1, "k1"))
		require.NoError(t, err)

		assert.Equal(t, int64(6000), ord.TotalCents)
		assert.Equal(t, int64(0), ord.ShippingCostCents)
	})

	t.Run("Retry with same key on consumed cart replays the same order", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser1, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		first, err := env.svc.CheckoutCart(t.Context(), featureCheckoutRequest(featureUser1, "k1"))
		require.NoError(t, err)
		second, err := env.svc.CheckoutCart(t.Context(), featureCheckoutRequest(featureUser1, "k1"))
		require.NoError(t, err)

		assert.Equal(t, first.ID, second.ID)

		stored := env.storedCart(t, featureUser1)
		assert.Equal(t, domain.StatusConsumed, stored.Status)
		assert.Equal(t, first.ID, stored.OrderID)
	})

	t.Run("Different key while checkout_pending returns checkout in progress", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.pendingCart(t, featureUser1, "k1", featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))

		_, err := env.svc.CheckoutCart(t.Context(), featureCheckoutRequest(featureUser1, "k2"))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCartCheckoutInProgress)
	})

	t.Run("Same key while checkout_pending retries the placement", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.pendingCart(t, featureUser1, "k1", featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))
		env.order.orders["k1"] = orderclient.Order{
			ID:         "ord-k1",
			UserID:     featureUser1,
			CustomerID: "cust-" + featureUser1,
			Status:     "paid",
			TotalCents: 1000,
		}

		ord, err := env.svc.CheckoutCart(t.Context(), featureCheckoutRequest(featureUser1, "k1"))
		require.NoError(t, err)

		assert.Equal(t, "ord-k1", ord.ID)
		assert.Equal(t, "k1", env.order.lastRequest.IdempotencyKey)
	})

	t.Run("Insufficient stock releases pending state back to active", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser1, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))
		env.order.stock[featureProductID] = 0

		_, err := env.svc.CheckoutCart(t.Context(), featureCheckoutRequest(featureUser1, "k1"))

		require.Error(t, err)
		require.ErrorIs(t, err, orderclient.ErrInsufficientStock)

		stored := env.storedCart(t, featureUser1)
		assert.Equal(t, domain.StatusActive, stored.Status)
	})

	t.Run("Mark-consumed update failure is non-fatal", func(t *testing.T) {
		env := newFeatureEnv(t)
		env.seedCart(t, featureUser1, 1, featureItem(featureProductID, featureProductName, featureProductPriceCents, 2))
		failing := &failingUpdateRepo{Repository: env.repo, failAt: 2}
		svc := NewService(failing, env.modules, metrics.Noop{})

		ord, err := svc.CheckoutCart(t.Context(), featureCheckoutRequest(featureUser1, "k1"))

		// The mark-consumed update is non-fatal: the placed order is returned
		// even though the cart could not be marked consumed.
		require.NoError(t, err)
		assert.Equal(t, "ord-k1", ord.ID)
		assert.Equal(t, 2, failing.calls)
	})
}
