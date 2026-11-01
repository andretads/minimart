package application

import (
	"minimart/internal/order/domain"
	"minimart/internal/order/infrastructure/memory"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOrders(t *testing.T) {
	newOrder := func(t *testing.T, repo *memory.Repository, id, userID string, createdAt time.Time) domain.Order {
		t.Helper()
		order, err := domain.NewOrder(id, userID, "", "", []domain.Item{
			{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
		})
		require.NoError(t, err)
		order.CreatedAt = createdAt
		require.NoError(t, repo.Save(t.Context(), order))
		return order
	}

	t.Run("returns empty page for user with no orders", func(t *testing.T) {
		svc := NewService(memory.New(), stubModules{}, noopIdem{}, metrics.Noop{}, time.Hour)

		got, err := svc.ListOrders(t.Context(), ListOrdersRequest{UserID: "u1", Page: 1, PageSize: 20})
		require.NoError(t, err)
		assert.Empty(t, got.Items)
		assert.Zero(t, got.Total)
		assert.Zero(t, got.TotalPages)
	})

	t.Run("returns only the requested user's orders newest first", func(t *testing.T) {
		repo := memory.New()
		now := time.Now().UTC()
		newOrder(t, repo, "o1", "u1", now.Add(-2*time.Hour))
		newOrder(t, repo, "o2", "u1", now.Add(-1*time.Hour))
		newOrder(t, repo, "o3", "u2", now)

		svc := NewService(repo, stubModules{}, noopIdem{}, metrics.Noop{}, time.Hour)
		got, err := svc.ListOrders(t.Context(), ListOrdersRequest{UserID: "u1", Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Len(t, got.Items, 2)
		assert.Equal(t, "o2", got.Items[0].ID)
		assert.Equal(t, "o1", got.Items[1].ID)
		assert.Equal(t, 2, got.Total)
		assert.Equal(t, 1, got.TotalPages)
	})

	t.Run("paginates and reports total pages", func(t *testing.T) {
		repo := memory.New()
		now := time.Now().UTC()
		newOrder(t, repo, "o1", "u1", now.Add(-4*time.Hour))
		newOrder(t, repo, "o2", "u1", now.Add(-3*time.Hour))
		newOrder(t, repo, "o3", "u1", now.Add(-2*time.Hour))
		newOrder(t, repo, "o4", "u1", now.Add(-1*time.Hour))

		svc := NewService(repo, stubModules{}, noopIdem{}, metrics.Noop{}, time.Hour)
		got, err := svc.ListOrders(t.Context(), ListOrdersRequest{UserID: "u1", Page: 2, PageSize: 3})
		require.NoError(t, err)
		require.Len(t, got.Items, 1)
		assert.Equal(t, "o1", got.Items[0].ID)
		assert.Equal(t, 4, got.Total)
		assert.Equal(t, 2, got.TotalPages)
	})

	t.Run("requires a user id", func(t *testing.T) {
		svc := NewService(memory.New(), stubModules{}, noopIdem{}, metrics.Noop{}, time.Hour)
		_, err := svc.ListOrders(t.Context(), ListOrdersRequest{UserID: " ", Page: 1, PageSize: 20})
		require.ErrorIs(t, err, domain.ErrUserIDRequired)
	})

	t.Run("rejects invalid pagination", func(t *testing.T) {
		svc := NewService(memory.New(), stubModules{}, noopIdem{}, metrics.Noop{}, time.Hour)

		_, err := svc.ListOrders(t.Context(), ListOrdersRequest{UserID: "u1", Page: 0, PageSize: 20})
		require.ErrorIs(t, err, domain.ErrInvalidPage)

		_, err = svc.ListOrders(t.Context(), ListOrdersRequest{UserID: "u1", Page: 1, PageSize: 0})
		require.ErrorIs(t, err, domain.ErrInvalidPageSize)

		_, err = svc.ListOrders(t.Context(), ListOrdersRequest{UserID: "u1", Page: 1, PageSize: 101})
		require.ErrorIs(t, err, domain.ErrInvalidPageSize)
	})
}
