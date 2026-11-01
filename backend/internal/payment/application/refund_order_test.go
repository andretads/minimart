package application

import (
	"errors"
	"minimart/internal/payment/domain"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefundOrderPayments(t *testing.T) {
	repo, idem, provider, m := newFakeRepo(), newFakeIdem(), approvedProvider(), &countingMetrics{}
	svc := NewService(repo, cancelledOrderReader(), idem, provider, m, time.Hour)
	succeededPayment(t, repo, "pay-1", "o1")
	succeededPayment(t, repo, "pay-2", "o1")
	succeededPayment(t, repo, "pay-other-order", "o2")

	require.NoError(t, svc.RefundOrderPayments(t.Context(), "o1"))

	for _, id := range []string{"pay-1", "pay-2"} {
		stored, err := repo.GetByID(t.Context(), id)
		require.NoError(t, err)
		assert.Equal(t, domain.StatusRefunded, stored.Status)
	}
	stored, err := repo.GetByID(t.Context(), "pay-other-order")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusSucceeded, stored.Status, "payments of other orders are untouched")
	assert.Equal(t, []string{"refunded", "refunded"}, m.payments)
}

func TestRefundOrderPaymentsSkipsAlreadyRefunded(t *testing.T) {
	repo, idem, provider := newFakeRepo(), newFakeIdem(), approvedProvider()
	svc := NewService(repo, cancelledOrderReader(), idem, provider, metrics.Noop{}, time.Hour)
	refunded := succeededPayment(t, repo, "pay-1", "o1")
	refunded, err := refunded.Refund(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-pay-1"})
	require.NoError(t, err)
	repo.byID["pay-1"] = refunded

	require.NoError(t, svc.RefundOrderPayments(t.Context(), "o1"))
	assert.Equal(t, 0, provider.refunded)
}

func TestRefundOrderPaymentsTreatsLostRaceAsNoOp(t *testing.T) {
	repo, idem, provider := newFakeRepo(), newFakeIdem(), approvedProvider()
	repo.refundErr = domain.ErrPaymentNotRefundable
	svc := NewService(repo, cancelledOrderReader(), idem, provider, metrics.Noop{}, time.Hour)
	succeededPayment(t, repo, "pay-1", "o1")

	require.NoError(t, svc.RefundOrderPayments(t.Context(), "o1"), "at-least-once delivery must not fail on a raced refund")
}

func TestRefundOrderPaymentsPropagatesProviderFailure(t *testing.T) {
	repo, idem := newFakeRepo(), newFakeIdem()
	provider := &stubProvider{refundErr: errors.New("gateway down")}
	svc := NewService(repo, cancelledOrderReader(), idem, provider, metrics.Noop{}, time.Hour)
	succeededPayment(t, repo, "pay-1", "o1")

	err := svc.RefundOrderPayments(t.Context(), "o1")
	require.ErrorIs(t, err, ErrRefundFailed)
}

func TestRefundOrderPaymentsUnknownOrderIsNoOp(t *testing.T) {
	repo, idem, provider := newFakeRepo(), newFakeIdem(), approvedProvider()
	svc := NewService(repo, nil, idem, provider, metrics.Noop{}, time.Hour)

	require.NoError(t, svc.RefundOrderPayments(t.Context(), "no-such-order"))
	assert.Equal(t, 0, provider.refunded)
}
