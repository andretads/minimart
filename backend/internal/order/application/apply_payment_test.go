package application

import (
	"minimart/internal/order/domain"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPaymentOutcome(t *testing.T) {
	t.Run("Order does not exist", func(t *testing.T) {
		repo := &fakeRepo{getErr: domain.ErrOrderNotFound}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		err := svc.ApplyPaymentOutcome(t.Context(), "missing", PaymentOutcomeSucceeded, "")
		require.ErrorIs(t, err, domain.ErrOrderNotFound)
		assert.Zero(t, repo.updateCalls)
	})

	t.Run("A duplicate delivery must not change anything", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t).MarkPaid()}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, ""))
		assert.Equal(t, domain.StatusPaid, repo.saved.Status)
		assert.Zero(t, repo.updateCalls, "no order write was attempted")
	})

	t.Run("A cancelled order must not be resurrected by a late payment", func(t *testing.T) {
		cancelled, err := placedOrder(t).Cancel()
		require.NoError(t, err)
		repo := &fakeRepo{saved: cancelled}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, ""))
		assert.Equal(t, domain.StatusCancelled, repo.saved.Status)
		assert.Zero(t, repo.updateCalls, "no order write was attempted")
	})

	t.Run("A failed payment must not be resurrected by a late success", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t).MarkPaymentFailed()}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, ""))
		assert.Equal(t, domain.StatusPaymentFailed, repo.saved.Status)
		assert.Zero(t, repo.updateCalls, "no order write was attempted")
	})

	t.Run("An unknown payment can still be resolved", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t).MarkPaymentUnknown()}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, ""))
		assert.Equal(t, domain.StatusPaid, repo.saved.Status)
		assert.Equal(t, domain.StatusPaymentUnknown, repo.updateExpected)
	})

	t.Run("Payment succeeded marks the order paid", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t)}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, ""))
		assert.Equal(t, domain.StatusPaid, repo.saved.Status)
		assert.Equal(t, domain.StatusPlaced, repo.updateExpected)
	})

	t.Run("Payment failed marks the order payment_failed and releases stock", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t)}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeFailed, ""))
		assert.Equal(t, domain.StatusPaymentFailed, repo.saved.Status)
		assert.Equal(t, "o1", findReleased(t, repo.saved).OrderID)
	})

	t.Run("Payment unknown marks the order payment_unknown and holds stock", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t)}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeUnknown, ""))
		assert.Equal(t, domain.StatusPaymentUnknown, repo.saved.Status)
		assert.Zero(t, countReleased(repo.saved), "an unresolved payment holds the reservation")
	})

	t.Run("An order without shipping hands nothing over to the shipping domain", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t)}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, ""))
		assert.Equal(t, domain.StatusPaid, repo.saved.Status)
		assert.Zero(t, countPaid(repo.saved), "an order without shipping has nothing to hand over")
	})

	t.Run("Payment succeeded on an order with shipping hands the snapshot over", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrderWithShipping(t)}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, ""))
		assert.Equal(t, domain.StatusPaid, repo.saved.Status)

		paid := findPaid(t, repo.saved)
		assert.Equal(t, "o1", paid.OrderID)
		assert.Equal(t, "q1", paid.ShippingQuoteID)
		assert.Equal(t, "m1", paid.ShippingMethodID)
		assert.Equal(t, featureDestination(), paid.DestinationSnapshot)
	})
}
