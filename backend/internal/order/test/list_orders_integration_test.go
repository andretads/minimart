//go:build integration

package test

import (
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOrdersIntegration(t *testing.T) {
	now := time.Now().UTC()

	t.Run("user id is missing", func(t *testing.T) {
		svc, _, _, _ := newOrderService(t)

		_, err := svc.ListOrders(t.Context(), application.ListOrdersRequest{Page: 1, PageSize: 20})
		require.ErrorIs(t, err, domain.ErrUserIDRequired)
	})

	t.Run("page is invalid", func(t *testing.T) {
		svc, _, _, _ := newOrderService(t)

		_, err := svc.ListOrders(t.Context(), application.ListOrdersRequest{UserID: uniqueOrderID(), Page: 0, PageSize: 20})
		require.ErrorIs(t, err, domain.ErrInvalidPage)
	})

	t.Run("page size is invalid", func(t *testing.T) {
		svc, _, _, _ := newOrderService(t)

		_, err := svc.ListOrders(t.Context(), application.ListOrdersRequest{UserID: uniqueOrderID(), Page: 1, PageSize: 101})
		require.ErrorIs(t, err, domain.ErrInvalidPageSize)
	})

	t.Run("user has no orders", func(t *testing.T) {
		svc, _, _, _ := newOrderService(t)

		got, err := svc.ListOrders(t.Context(), application.ListOrdersRequest{
			UserID: uniqueOrderID(), Page: 1, PageSize: 20,
		})
		require.NoError(t, err)
		assert.Empty(t, got.Items)
		assert.Zero(t, got.Total)
		assert.Zero(t, got.TotalPages)
	})

	t.Run("user's orders are listed newest first", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		userID := uniqueOrderID()
		older := uniqueOrderID()
		newer := uniqueOrderID()
		seedOrder(t, repo, pool, older, userID)
		seedOrder(t, repo, pool, newer, userID)
		setCreatedAt(t, pool, older, now.Add(-1*time.Hour))
		setCreatedAt(t, pool, newer, now)

		got, err := svc.ListOrders(t.Context(), application.ListOrdersRequest{UserID: userID, Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Len(t, got.Items, 2)
		assert.Equal(t, newer, got.Items[0].ID)
		assert.Equal(t, older, got.Items[1].ID)
		for _, o := range got.Items {
			require.Len(t, o.Items, 1)
			assert.Equal(t, "p1", o.Items[0].ProductID)
			assert.Equal(t, 1, o.Items[0].Quantity)
		}
	})

	t.Run("pagination returns the requested window and metadata", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		userID := uniqueOrderID()
		oldest := uniqueOrderID()
		middle := uniqueOrderID()
		newest := uniqueOrderID()
		seedOrder(t, repo, pool, oldest, userID)
		seedOrder(t, repo, pool, middle, userID)
		seedOrder(t, repo, pool, newest, userID)
		setCreatedAt(t, pool, oldest, now.Add(-3*time.Hour))
		setCreatedAt(t, pool, middle, now.Add(-2*time.Hour))
		setCreatedAt(t, pool, newest, now.Add(-1*time.Hour))

		got, err := svc.ListOrders(t.Context(), application.ListOrdersRequest{UserID: userID, Page: 2, PageSize: 2})
		require.NoError(t, err)
		require.Len(t, got.Items, 1)
		assert.Equal(t, oldest, got.Items[0].ID)
		assert.Equal(t, 2, got.TotalPages)
	})

	t.Run("another user's orders are never returned", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		userID := uniqueOrderID()
		otherUserID := uniqueOrderID()
		seedOrder(t, repo, pool, uniqueOrderID(), userID)
		seedOrder(t, repo, pool, uniqueOrderID(), otherUserID)

		got, err := svc.ListOrders(t.Context(), application.ListOrdersRequest{UserID: userID, Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Len(t, got.Items, 1)
		assert.Equal(t, userID, got.Items[0].UserID)
	})

	t.Run("Placed orders in a list expose their auto-cancel deadline", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		userID := uniqueOrderID()
		seedOrder(t, repo, pool, uniqueOrderID(), userID)

		got, err := svc.ListOrders(t.Context(), application.ListOrdersRequest{UserID: userID, Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Len(t, got.Items, 1)
		require.NotNil(t, got.Items[0].AutoCancelAt())
	})
}
