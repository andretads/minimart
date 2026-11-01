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

func TestAutoCancelStaleOrders(t *testing.T) {
	stale := func(t *testing.T, repo *memory.Repository, id string) domain.Order {
		t.Helper()
		return saveOrder(t, repo, newStoredOrder(t, id, "u1", domain.StatusPlaced, time.Now().UTC().Add(-6*time.Minute)))
	}

	t.Run("Stale placed orders are cancelled", func(t *testing.T) {
		repo := memory.New()
		stale(t, repo, "o1")

		NewAutoCanceller(repo, metrics.Noop{}, 10, nil).RunOnce(t.Context())

		got, err := repo.GetByID(t.Context(), "o1")
		require.NoError(t, err)
		assert.Equal(t, domain.StatusCancelled, got.Status)
		assert.Equal(t, "o1", findCancelled(t, got).OrderID)
		assert.Equal(t, "o1", findReleased(t, got).OrderID)
	})

	t.Run("Fresh placed orders are left alone", func(t *testing.T) {
		repo := memory.New()
		saveOrder(t, repo, newStoredOrder(t, "o2", "u1", domain.StatusPlaced, time.Now().UTC().Add(-1*time.Minute)))

		NewAutoCanceller(repo, metrics.Noop{}, 10, nil).RunOnce(t.Context())

		got, err := repo.GetByID(t.Context(), "o2")
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPlaced, got.Status)
	})

	t.Run("Concurrent status change is not clobbered", func(t *testing.T) {
		repo := memory.New()
		stale(t, repo, "o3")

		placed, err := repo.GetByID(t.Context(), "o3")
		require.NoError(t, err)
		require.NoError(t, repo.Update(t.Context(), placed.MarkPaid(), domain.StatusPlaced))

		race := &stalePlacedRepo{Repository: repo, stale: []domain.Order{placed}}
		NewAutoCanceller(race, metrics.Noop{}, 10, nil).RunOnce(t.Context())

		got, err := repo.GetByID(t.Context(), "o3")
		require.NoError(t, err)
		assert.Equal(t, domain.StatusPaid, got.Status, "the concurrent transition won and must be kept")
	})
}
