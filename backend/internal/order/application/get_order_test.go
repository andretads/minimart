package application

import (
	"minimart/internal/order/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrder(t *testing.T) {
	t.Run("Order does not exist", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.GetOrder(t.Context(), "missing")
		require.ErrorIs(t, err, domain.ErrOrderNotFound)
	})

	t.Run("Order got returned with no problems", func(t *testing.T) {
		svc, repo := newTestService(t)
		saveOrder(t, repo, newStoredOrder(t, "o1", "u1", domain.StatusPlaced, time.Now().UTC()))

		got, err := svc.GetOrder(t.Context(), "o1")
		require.NoError(t, err)
		assert.Equal(t, "o1", got.ID)
		assert.Equal(t, "u1", got.UserID)
		assert.Equal(t, int64(500), got.TotalCents)
	})

	t.Run("The shipping snapshot of the order is returned", func(t *testing.T) {
		svc, repo := newTestService(t)
		o := newStoredOrder(t, "o1", "u1", domain.StatusPlaced, time.Now().UTC())
		o.WithShipping(domain.ShippingSnapshot{
			QuoteID:     "q1",
			MethodID:    "m1",
			MethodName:  "Standard",
			CostCents:   250,
			Destination: featureDestination(),
		})
		saveOrder(t, repo, o)

		got, err := svc.GetOrder(t.Context(), "o1")
		require.NoError(t, err)
		assert.Equal(t, "o1", got.ID)
		assert.Equal(t, int64(750), got.TotalCents)
		assert.Equal(t, "q1", got.ShippingQuoteID)
		assert.Equal(t, "m1", got.ShippingMethodID)
		assert.Equal(t, int64(250), got.ShippingCostCents)
		assert.Equal(t, featureDestination(), got.DestinationSnapshot)
	})

	t.Run("A placed order exposes its auto-cancel deadline", func(t *testing.T) {
		svc, repo := newTestService(t)
		saveOrder(t, repo, newStoredOrder(t, "o1", "u1", domain.StatusPlaced, time.Now().UTC()))

		got, err := svc.GetOrder(t.Context(), "o1")
		require.NoError(t, err)
		require.NotNil(t, got.AutoCancelAt(), "a placed order has an auto-cancel deadline")
		assert.Equal(t, got.CreatedAt.Add(domain.AutoCancelAfter), *got.AutoCancelAt())
	})
}
