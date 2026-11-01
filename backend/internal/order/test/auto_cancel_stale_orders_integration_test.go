//go:build integration

package test

import (
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoCancelStaleOrdersIntegration(t *testing.T) {
	t.Run("Stale placed orders are cancelled", func(t *testing.T) {
		repo, pool := newOrderRepo(t)
		id := uniqueOrderID()
		seedOrder(t, repo, pool, id, "it-user")
		setCreatedAt(t, pool, id, time.Now().UTC().Add(-6*time.Minute))

		application.NewAutoCanceller(repo, metrics.Noop{}, 100, nil).RunOnce(t.Context())

		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusCancelled, got.Status)
		assert.Equal(t, 1, countOutboxTopic(t.Context(), t, pool, id, kafka.OrderCancelledTopic.String()))
		assert.Equal(t, 1, countOutboxTopic(t.Context(), t, pool, id, kafka.OrderStockReleaseTopic.String()))
	})

	t.Run("Fresh placed orders are left alone", func(t *testing.T) {
		repo, pool := newOrderRepo(t)
		id := uniqueOrderID()
		seedOrder(t, repo, pool, id, "it-user")
		setCreatedAt(t, pool, id, time.Now().UTC().Add(-1*time.Minute))

		application.NewAutoCanceller(repo, metrics.Noop{}, 100, nil).RunOnce(t.Context())

		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPlaced, got.Status)
	})

	t.Run("Concurrent status change is not clobbered", func(t *testing.T) {
		repo, pool := newOrderRepo(t)
		id := uniqueOrderID()
		seedOrder(t, repo, pool, id, "it-user")
		setCreatedAt(t, pool, id, time.Now().UTC().Add(-6*time.Minute))

		placed, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		require.NoError(t, repo.Update(t.Context(), placed.MarkPaid(), domain.StatusPlaced))

		err = repo.Update(t.Context(), placed.MarkPaymentFailed(), domain.StatusPlaced)
		require.ErrorIs(t, err, domain.ErrOrderStateChanged, "the losing write is rejected")

		race := &staleListRepo{OrderRepository: repo, stale: []domain.Order{placed}}
		application.NewAutoCanceller(race, metrics.Noop{}, 100, nil).RunOnce(t.Context())

		got, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPaid, got.Status, "the concurrent transition won and must be kept")
		assert.Zero(t, countOutboxTopic(t.Context(), t, pool, id, kafka.OrderCancelledTopic.String()))
	})
}
