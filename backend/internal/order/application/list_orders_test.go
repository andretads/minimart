package application

import (
	"minimart/internal/order/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOrders(t *testing.T) {
	now := time.Now().UTC()

	t.Run("user id is missing", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.ListOrders(t.Context(), ListOrdersRequest{Page: 1, PageSize: 20})
		require.ErrorIs(t, err, domain.ErrUserIDRequired)
	})

	t.Run("page is invalid", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.ListOrders(t.Context(), ListOrdersRequest{UserID: "u1", Page: 0, PageSize: 20})
		require.ErrorIs(t, err, domain.ErrInvalidPage)
	})

	t.Run("page size is invalid", func(t *testing.T) {
		svc, _ := newTestService(t)

		_, err := svc.ListOrders(t.Context(), ListOrdersRequest{UserID: "u1", Page: 1, PageSize: 101})
		require.ErrorIs(t, err, domain.ErrInvalidPageSize)
	})

	t.Run("user has no orders", func(t *testing.T) {
		svc, _ := newTestService(t)

		got, err := svc.ListOrders(t.Context(), ListOrdersRequest{UserID: "u1", Page: 1, PageSize: 20})
		require.NoError(t, err)
		assert.Empty(t, got.Items)
		assert.Zero(t, got.Total)
		assert.Zero(t, got.TotalPages)
	})

	t.Run("user's orders are listed newest first", func(t *testing.T) {
		svc, repo := newTestService(t)
		saveOrder(t, repo, newStoredOrder(t, "o1", "u1", domain.StatusPlaced, now.Add(-1*time.Hour)))
		saveOrder(t, repo, newStoredOrder(t, "o2", "u1", domain.StatusPlaced, now))

		got, err := svc.ListOrders(t.Context(), ListOrdersRequest{UserID: "u1", Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Len(t, got.Items, 2)
		assert.Equal(t, "o2", got.Items[0].ID)
		assert.Equal(t, "o1", got.Items[1].ID)
	})

	t.Run("pagination returns the requested window and metadata", func(t *testing.T) {
		svc, repo := newTestService(t)
		saveOrder(t, repo, newStoredOrder(t, "o1", "u1", domain.StatusPlaced, now.Add(-3*time.Hour)))
		saveOrder(t, repo, newStoredOrder(t, "o2", "u1", domain.StatusPlaced, now.Add(-2*time.Hour)))
		saveOrder(t, repo, newStoredOrder(t, "o3", "u1", domain.StatusPlaced, now.Add(-1*time.Hour)))

		got, err := svc.ListOrders(t.Context(), ListOrdersRequest{UserID: "u1", Page: 2, PageSize: 2})
		require.NoError(t, err)
		require.Len(t, got.Items, 1)
		assert.Equal(t, "o1", got.Items[0].ID)
		assert.Equal(t, 2, got.TotalPages)
	})

	t.Run("another user's orders are never returned", func(t *testing.T) {
		svc, repo := newTestService(t)
		saveOrder(t, repo, newStoredOrder(t, "o1", "u1", domain.StatusPlaced, now.Add(-1*time.Hour)))
		saveOrder(t, repo, newStoredOrder(t, "o2", "u2", domain.StatusPlaced, now))

		got, err := svc.ListOrders(t.Context(), ListOrdersRequest{UserID: "u1", Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Len(t, got.Items, 1)
		assert.Equal(t, "u1", got.Items[0].UserID)
	})

	t.Run("Placed orders in a list expose their auto-cancel deadline", func(t *testing.T) {
		svc, repo := newTestService(t)
		saveOrder(t, repo, newStoredOrder(t, "o1", "u1", domain.StatusPlaced, now))

		got, err := svc.ListOrders(t.Context(), ListOrdersRequest{UserID: "u1", Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Len(t, got.Items, 1)
		require.NotNil(t, got.Items[0].AutoCancelAt())
	})
}
