package application

import (
	"errors"
	"minimart/internal/order/domain"
	"minimart/pkg/metrics"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPaymentOutcome(t *testing.T) {
	tests := []struct {
		name    string
		outcome PaymentOutcome
		want    domain.Status
	}{
		{"succeeded pays the order", PaymentOutcomeSucceeded, domain.StatusPaid},
		{"failed fails the payment", PaymentOutcomeFailed, domain.StatusPaymentFailed},
		{"unknown leaves the payment unresolved", PaymentOutcomeUnknown, domain.StatusPaymentUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{saved: placedOrder(t)}
			m := &countingMetrics{}
			svc := newSvc(repo, testCatalog(), m)

			require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", tt.outcome, "reason"))
			assert.Equal(t, tt.want, repo.saved.Status)
			assert.Equal(t, domain.StatusPlaced, repo.updateExpected, "the CAS guard uses the status read before the update")
			assert.Equal(t, 1, repo.updateCalls)
			assert.Equal(t, []string{metrics.StatusSuccess}, m.paymentApplied)
		})
	}
}

func TestApplyPaymentOutcomeIsIdempotent(t *testing.T) {
	t.Run("redelivering succeeded to a paid order writes nothing", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t).MarkPaid()}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, ""))
		assert.Equal(t, 0, repo.updateCalls)
	})

	t.Run("a cancelled order is not resurrected", func(t *testing.T) {
		cancelled, err := placedOrder(t).Cancel()
		require.NoError(t, err)

		repo := &fakeRepo{saved: cancelled}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, ""))
		assert.Equal(t, 0, repo.updateCalls)
		assert.Equal(t, domain.StatusCancelled, repo.saved.Status)
	})

	t.Run("an unknown payment can still be resolved to paid", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t).MarkPaymentUnknown()}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, ""))
		assert.Equal(t, domain.StatusPaid, repo.saved.Status)
		assert.Equal(t, domain.StatusPaymentUnknown, repo.updateExpected)
	})
}

func TestApplyPaymentOutcomeLostRaceIsBenign(t *testing.T) {
	repo := &fakeRepo{saved: placedOrder(t), updateErr: domain.ErrOrderStateChanged}
	m := &countingMetrics{}
	svc := newSvc(repo, testCatalog(), m)

	require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, ""))
	assert.Equal(t, []string{metrics.StatusSuccess}, m.paymentApplied)
}

func TestApplyPaymentOutcomeShippingHandoff(t *testing.T) {
	t.Run("an order with shipping raises the order-paid event", func(t *testing.T) {
		o := placedOrder(t)
		o.ShippingQuoteID = "quote-1"
		o.ShippingMethodID = "method-1"
		o.ShippingMethodName = "Standard"
		o.ShippingCostCents = 250
		o.DestinationSnapshot = domain.DestinationSnapshot{
			FullName: "Ada Lovelace",
			Street:   "1 Main St",
			City:     "Testville",
			Zip:      "12345",
			Country:  "BR",
		}
		repo := &fakeRepo{saved: o}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, ""))
		assert.Equal(t, domain.StatusPaid, repo.saved.Status)

		paid := findPaid(t, repo.saved)
		assert.Equal(t, "quote-1", paid.ShippingQuoteID)
		assert.Equal(t, "method-1", paid.ShippingMethodID)
		assert.Equal(t, o.DestinationSnapshot, paid.DestinationSnapshot)
	})

	t.Run("an order without shipping raises no order-paid event", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t)}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		require.NoError(t, svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, ""))
		assert.Equal(t, domain.StatusPaid, repo.saved.Status)

		for _, ev := range repo.saved.Events() {
			_, isPaid := ev.(domain.OrderPaidEvent)
			assert.False(t, isPaid, "an order without shipping has nothing to hand over")
		}
	})
}

// findPaid returns the OrderPaidEvent recorded on the aggregate, which the
// repository would write to the outbox for the shipping domain to consume.
func findPaid(t *testing.T, o domain.Order) domain.OrderPaidEvent {
	t.Helper()
	for _, ev := range o.Events() {
		if e, ok := ev.(domain.OrderPaidEvent); ok {
			return e
		}
	}
	t.Fatal("no OrderPaidEvent recorded on the order")
	return domain.OrderPaidEvent{}
}

func TestApplyPaymentOutcomeErrors(t *testing.T) {
	t.Run("missing order propagates not found", func(t *testing.T) {
		repo := &fakeRepo{getErr: domain.ErrOrderNotFound}
		m := &countingMetrics{}
		svc := newSvc(repo, testCatalog(), m)

		err := svc.ApplyPaymentOutcome(t.Context(), "missing", PaymentOutcomeSucceeded, "")
		require.ErrorIs(t, err, domain.ErrOrderNotFound)
		assert.Equal(t, []string{metrics.StatusError}, m.paymentApplied)
	})

	t.Run("unknown outcome is rejected", func(t *testing.T) {
		repo := &fakeRepo{saved: placedOrder(t)}
		svc := newSvc(repo, testCatalog(), metrics.Noop{})

		err := svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcome("bogus"), "")
		require.Error(t, err)
		assert.Equal(t, 0, repo.updateCalls)
	})

	t.Run("persistence failure propagates", func(t *testing.T) {
		boom := errors.New("disk on fire")
		repo := &fakeRepo{saved: placedOrder(t), updateErr: boom}
		m := &countingMetrics{}
		svc := newSvc(repo, testCatalog(), m)

		err := svc.ApplyPaymentOutcome(t.Context(), "o1", PaymentOutcomeSucceeded, "")
		require.ErrorIs(t, err, boom)
		assert.Equal(t, []string{metrics.StatusError}, m.paymentApplied)
	})
}
