package application

import (
	"errors"
	"minimart/internal/order/domain"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelOrder(t *testing.T) {
	t.Run("Order does not exist", func(t *testing.T) {
		repo := &fakeRepo{getErr: domain.ErrOrderNotFound}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		_, err := svc.CancelOrder(t.Context(), "missing")
		require.ErrorIs(t, err, domain.ErrOrderNotFound)
	})

	t.Run("Order is already cancelled", func(t *testing.T) {
		cancelled, err := placedOrder(t).Cancel()
		require.NoError(t, err)
		repo := &fakeRepo{saved: cancelled}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		_, err = svc.CancelOrder(t.Context(), "o1")
		require.ErrorIs(t, err, domain.ErrOrderAlreadyCancelled)
	})

	t.Run("Order is paid", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t).MarkPaid()}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		_, err := svc.CancelOrder(t.Context(), "o1")
		require.ErrorIs(t, err, domain.ErrOrderNotCancellable)
		assert.Zero(t, repo.updateCalls, "no order write was attempted")
	})

	t.Run("Storage failure", func(t *testing.T) {
		boom := errors.New("disk on fire")
		repo := &fakeRepo{saved: placedOrder(t), updateErr: boom}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		_, err := svc.CancelOrder(t.Context(), "o1")
		require.ErrorIs(t, err, boom)
		assert.Zero(t, countCancelled(repo.saved), "no OrderCancelled event will be published")
	})

	t.Run("Order got cancelled and its stock release is outboxed", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t)}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		got, err := svc.CancelOrder(t.Context(), "o1")
		require.NoError(t, err)

		assert.Equal(t, "u1", repo.saved.UserID)
		assert.Equal(t, domain.StatusCancelled, repo.saved.Status)
		assert.Equal(t, got.Status, repo.saved.Status)
		require.NotNil(t, repo.saved.CancelledAt)

		assert.Equal(t, "o1", findCancelled(t, repo.saved).OrderID)
		assert.Equal(t, "o1", findReleased(t, repo.saved).OrderID)
	})
}
