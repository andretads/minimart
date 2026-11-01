//go:build integration

package test

import (
	"minimart/internal/order/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrderIntegration(t *testing.T) {
	t.Run("Order does not exist", func(t *testing.T) {
		svc, _, _, _ := newOrderService(t)

		_, err := svc.GetOrder(t.Context(), uniqueOrderID())
		require.ErrorIs(t, err, domain.ErrOrderNotFound)
	})

	t.Run("Order got returned with no problems", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedOrder(t, repo, pool, id, "it-user")

		got, err := svc.GetOrder(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, id, got.ID)
		assert.Equal(t, "it-user", got.UserID)
		assert.Equal(t, int64(500), got.TotalCents)
		require.Len(t, got.Items, 1)
		assert.Equal(t, "p1", got.Items[0].ProductID)
		assert.Equal(t, 1, got.Items[0].Quantity)
		assert.False(t, got.CreatedAt.IsZero())
		assert.WithinDuration(t, time.Now().UTC(), got.CreatedAt, time.Minute)
	})

	t.Run("The shipping snapshot of the order is returned", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedShippingOrder(t, repo, pool, id, "it-user")

		got, err := svc.GetOrder(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, id, got.ID)
		assert.Equal(t, int64(750), got.TotalCents)
		assert.Equal(t, "q1", got.ShippingQuoteID)
		assert.Equal(t, "m1", got.ShippingMethodID)
		assert.Equal(t, int64(250), got.ShippingCostCents)
		assert.Equal(t, featureDestination(), got.DestinationSnapshot)
	})

	t.Run("A placed order exposes its auto-cancel deadline", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedOrder(t, repo, pool, id, "it-user")

		got, err := svc.GetOrder(t.Context(), id)
		require.NoError(t, err)
		require.NotNil(t, got.AutoCancelAt())
		assert.Equal(t, got.CreatedAt.Add(domain.AutoCancelAfter), *got.AutoCancelAt())
	})
}
