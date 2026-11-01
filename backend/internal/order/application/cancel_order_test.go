package application

import (
	"errors"
	"minimart/internal/order/domain"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func placedOrder(t *testing.T) domain.Order {
	t.Helper()
	o, err := domain.NewOrder("o1", "u1", "c1", "", []domain.Item{
		{ProductID: "p1", Name: "Widget", UnitPriceCents: 500, Quantity: 1},
	})
	require.NoError(t, err)
	return o
}

// findCancelled returns the OrderCancelledEvent recorded on the aggregate, which
// the repository would write to the outbox.
func findCancelled(t *testing.T, o domain.Order) domain.OrderCancelledEvent {
	t.Helper()
	for _, ev := range o.Events() {
		if e, ok := ev.(domain.OrderCancelledEvent); ok {
			return e
		}
	}
	t.Fatal("no OrderCancelledEvent recorded on the order")
	return domain.OrderCancelledEvent{}
}

// findReleased returns the OrderStockReleaseEvent recorded on the aggregate, which
// the repository would write to the outbox and the catalog would consume.
func findReleased(t *testing.T, o domain.Order) domain.OrderStockReleaseEvent {
	t.Helper()
	for _, ev := range o.Events() {
		if e, ok := ev.(domain.OrderStockReleaseEvent); ok {
			return e
		}
	}
	t.Fatal("no OrderStockReleaseEvent recorded on the order")
	return domain.OrderStockReleaseEvent{}
}

func TestCancelOrder(t *testing.T) {
	t.Run("cancels a placed order and records one event", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t)}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		got, err := svc.CancelOrder(t.Context(), "o1")
		require.NoError(t, err)
		assert.Equal(t, domain.StatusCancelled, got.Status)
		require.NotNil(t, got.CancelledAt)
		assert.Equal(t, domain.StatusCancelled, repo.saved.Status, "the cancelled state is persisted")

		event := findCancelled(t, repo.saved)
		assert.Equal(t, "o1", event.OrderID)
		assert.Equal(t, "u1", event.UserID)
		assert.Equal(t, int64(500), event.TotalCents)
		assert.Equal(t, 1, event.ItemCount)

		release := findReleased(t, repo.saved)
		assert.Equal(t, "o1", release.OrderID, "cancelling releases the order's stock reservation")
	})

	t.Run("missing order returns not found", func(t *testing.T) {
		repo := &fakeRepo{getErr: domain.ErrOrderNotFound}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		_, err := svc.CancelOrder(t.Context(), "missing")
		require.ErrorIs(t, err, domain.ErrOrderNotFound)
	})

	t.Run("already cancelled order is rejected", func(t *testing.T) {
		o := placedOrder(t)
		cancelled, err := o.Cancel()
		require.NoError(t, err)

		repo := &fakeRepo{saved: cancelled}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		_, err = svc.CancelOrder(t.Context(), "o1")
		require.ErrorIs(t, err, domain.ErrOrderAlreadyCancelled)
	})

	t.Run("a paid order cannot be cancelled", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t).MarkPaid()}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		_, err := svc.CancelOrder(t.Context(), "o1")
		require.ErrorIs(t, err, domain.ErrOrderNotCancellable)
		assert.Equal(t, 0, repo.updateCalls, "a rejected cancellation writes nothing")
	})
}

func TestCancelOrderUpdateFailure(t *testing.T) {
	boom := errors.New("disk on fire")
	repo := &fakeRepo{saved: placedOrder(t), updateErr: boom}
	m := &countingMetrics{}
	svc := newSvc(repo, testCatalog(), m)

	_, err := svc.CancelOrder(t.Context(), "o1")
	require.ErrorIs(t, err, boom)
	assert.Equal(t, []string{metrics.StatusError}, m.cancelled)
}

func TestCancelOrderRecordsMetrics(t *testing.T) {
	m := &countingMetrics{}
	svc := newSvc(&fakeRepo{saved: placedOrder(t)}, testCatalog(), m)

	_, err := svc.CancelOrder(t.Context(), "o1")
	require.NoError(t, err)
	assert.Equal(t, []string{metrics.StatusSuccess}, m.cancelled)
}
