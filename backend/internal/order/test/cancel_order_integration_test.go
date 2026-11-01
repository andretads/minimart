//go:build integration

package test

import (
	"errors"
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelOrderIntegration(t *testing.T) {
	t.Run("Order does not exist", func(t *testing.T) {
		svc, repo, _, _ := newOrderService(t)
		missing := uniqueOrderID()

		_, err := svc.CancelOrder(t.Context(), missing)
		require.ErrorIs(t, err, domain.ErrOrderNotFound)

		order, err := domain.NewOrder(missing, "it-user", "", "", []domain.Item{
			{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
		})
		require.NoError(t, err)
		assert.ErrorIs(t, repo.Update(t.Context(), order, domain.StatusPlaced), domain.ErrOrderNotFound,
			"the storage guard rejects the missing order")
	})

	t.Run("Order is already cancelled", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedOrderWithStatus(t, repo, pool, id, "it-user", domain.StatusCancelled)

		_, err := svc.CancelOrder(t.Context(), id)
		require.ErrorIs(t, err, domain.ErrOrderAlreadyCancelled)
	})

	t.Run("Order is paid", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedOrderWithStatus(t, repo, pool, id, "it-user", domain.StatusPaid)
		before := countOutbox(t.Context(), t, pool, id)

		_, err := svc.CancelOrder(t.Context(), id)
		require.ErrorIs(t, err, domain.ErrOrderNotCancellable)

		assert.Equal(t, before, countOutbox(t.Context(), t, pool, id), "no order write was attempted")
		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPaid, got.Status)
	})

	t.Run("Storage failure", func(t *testing.T) {
		repo, pool := newOrderRepo(t)
		id := uniqueOrderID()
		seedOrder(t, repo, pool, id, "it-user")
		boom := errors.New("disk on fire")
		svc := application.NewService(failingUpdateRepo{OrderRepository: repo, err: boom}, nil, nil, metrics.Noop{}, 0)

		_, err := svc.CancelOrder(t.Context(), id)
		require.ErrorIs(t, err, boom)
		assert.Zero(t, countOutboxTopic(t.Context(), t, pool, id, kafka.OrderCancelledTopic.String()))

		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPlaced, got.Status)
	})

	t.Run("Order got cancelled and its stock release is outboxed", func(t *testing.T) {
		svc, repo, pool, _ := newOrderService(t)
		id := uniqueOrderID()
		seedOrder(t, repo, pool, id, "it-user")

		before, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPlaced, before.Status)
		assert.Nil(t, before.CancelledAt)
		clearOutbox(t, pool, id)

		got, err := svc.CancelOrder(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusCancelled, got.Status)

		stored, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, "it-user", stored.UserID)
		assert.Equal(t, domain.StatusCancelled, stored.Status)
		require.NotNil(t, stored.CancelledAt)
		assert.WithinDuration(t, *got.CancelledAt, *stored.CancelledAt, time.Second)

		assert.ElementsMatch(t, []string{
			kafka.OrderCancelledTopic.String(),
			kafka.OrderStockReleaseTopic.String(),
		}, outboxTopics(t.Context(), t, pool, id))
	})
}
