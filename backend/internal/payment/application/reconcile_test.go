package application

import (
	"errors"
	"minimart/internal/payment/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// storedUnknownPayment stores a payment whose capture outcome is unknown but
// that carries a charge reference.
func storedUnknownPayment(t *testing.T, repo domain.Repository, id, orderID string) domain.Payment {
	t.Helper()
	p, err := domain.NewPayment(id, orderID, "u1", "idem-"+id, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	p, err = p.ApplyOutcome(domain.Result{Outcome: domain.OutcomeTimeout, Reason: "slow", ProviderRef: "ref-" + id})
	require.NoError(t, err)
	p.UpdatedAt = time.Now().Add(-time.Hour)
	require.NoError(t, repo.Save(t.Context(), p))
	return p
}

func TestReconcilerRefundsCapturedChargeOnCancelledOrder(t *testing.T) {
	e := newTestEnv(t)
	m := &countingMetrics{}
	e.m = m
	e.useOrder(cancelledOrder())
	storedUnknownPayment(t, e.repo, "pay-1", "o1")

	NewReconciler(e.svc, time.Minute, 10, nil).RunOnce(t.Context())

	stored := storedPayment(t, e.repo, "pay-1")
	assert.Equal(t, domain.StatusRefunded, stored.Status, "a captured charge on a cancelled order is reversed, not finalized")
	assert.Equal(t, 1, e.provider.refunded)
	assert.Equal(t, []string{"refunded"}, m.payments)
}

func TestReconcilerFinalizesCapturedChargeOnPayableOrder(t *testing.T) {
	e := newTestEnv(t)
	m := &countingMetrics{}
	e.m = m
	e.useOrder(defaultOrder())
	storedUnknownPayment(t, e.repo, "pay-1", "o1")

	NewReconciler(e.svc, time.Minute, 10, nil).RunOnce(t.Context())

	stored := storedPayment(t, e.repo, "pay-1")
	assert.Equal(t, domain.StatusSucceeded, stored.Status)
	assert.Equal(t, 0, e.provider.refunded)
	assert.Equal(t, []string{"succeeded"}, m.payments)
}

func TestReconcilerFailsOpenWhenOrderUnreadable(t *testing.T) {
	e := newTestEnv(t)
	e.reader = &stubOrderReader{err: errors.New("order domain down")}
	e.wire()
	storedUnknownPayment(t, e.repo, "pay-1", "o1")

	NewReconciler(e.svc, time.Minute, 10, nil).RunOnce(t.Context())

	stored := storedPayment(t, e.repo, "pay-1")
	assert.Equal(t, domain.StatusSucceeded, stored.Status, "an unreadable order must not strand the payment")
	assert.Equal(t, 0, e.provider.refunded)
}

func TestReconcilerTreatsLostRefundRaceAsBenign(t *testing.T) {
	e := newTestEnv(t)
	e.useOrder(cancelledOrder())
	e.repo = &failingRepository{Repository: e.repo, refundErr: domain.ErrPaymentNotRefundable}
	e.wire()
	storedUnknownPayment(t, e.repo, "pay-1", "o1")

	NewReconciler(e.svc, time.Minute, 10, nil).RunOnce(t.Context())

	stored := storedPayment(t, e.repo, "pay-1")
	assert.Equal(t, domain.StatusUnknown, stored.Status, "a raced refund leaves the payment for the next pass")
}
