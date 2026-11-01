package application

import (
	"errors"
	"minimart/internal/order/presentation/module/client"
	"minimart/internal/payment/domain"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// storedUnknownPayment stores a payment whose capture outcome is unknown but
// that carries a charge reference.
func storedUnknownPayment(t *testing.T, repo *fakeRepo, id, orderID string) domain.Payment {
	t.Helper()
	p, err := domain.NewPayment(id, orderID, "u1", "idem-"+id, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	p, err = p.ApplyOutcome(domain.Result{Outcome: domain.OutcomeTimeout, Reason: "slow", ProviderRef: "ref-" + id})
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))
	return p
}

func TestReconcilerRefundsCapturedChargeOnCancelledOrder(t *testing.T) {
	repo, idem, provider, m := newFakeRepo(), newFakeIdem(), approvedProvider(), &countingMetrics{}
	svc := NewService(repo, cancelledOrderReader(), idem, provider, m, time.Hour)
	storedUnknownPayment(t, repo, "pay-1", "o1")

	NewReconciler(svc, time.Minute, 10, nil).RunOnce(t.Context())

	stored, err := repo.GetByID(t.Context(), "pay-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusRefunded, stored.Status, "a captured charge on a cancelled order is reversed, not finalized")
	assert.Equal(t, 1, provider.refunded)
	assert.Equal(t, []string{"refunded"}, m.payments)
}

func TestReconcilerFinalizesCapturedChargeOnPayableOrder(t *testing.T) {
	repo, idem, provider, m := newFakeRepo(), newFakeIdem(), approvedProvider(), &countingMetrics{}
	reader := &stubOrderReader{order: client.Order{ID: "o1", UserID: "u1", Status: "placed"}}
	svc := NewService(repo, reader, idem, provider, m, time.Hour)
	storedUnknownPayment(t, repo, "pay-1", "o1")

	NewReconciler(svc, time.Minute, 10, nil).RunOnce(t.Context())

	stored, err := repo.GetByID(t.Context(), "pay-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusSucceeded, stored.Status)
	assert.Equal(t, 0, provider.refunded)
	assert.Equal(t, []string{"succeeded"}, m.payments)
}

func TestReconcilerFailsOpenWhenOrderUnreadable(t *testing.T) {
	repo, idem, provider, m := newFakeRepo(), newFakeIdem(), approvedProvider(), &countingMetrics{}
	reader := &stubOrderReader{err: errors.New("order domain down")}
	svc := NewService(repo, reader, idem, provider, m, time.Hour)
	storedUnknownPayment(t, repo, "pay-1", "o1")

	NewReconciler(svc, time.Minute, 10, nil).RunOnce(t.Context())

	stored, err := repo.GetByID(t.Context(), "pay-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusSucceeded, stored.Status, "an unreadable order must not strand the payment")
	assert.Equal(t, 0, provider.refunded)
}

func TestReconcilerTreatsLostRefundRaceAsBenign(t *testing.T) {
	repo, idem, provider := newFakeRepo(), newFakeIdem(), approvedProvider()
	repo.refundErr = domain.ErrPaymentNotRefundable
	svc := NewService(repo, cancelledOrderReader(), idem, provider, metrics.Noop{}, time.Hour)
	storedUnknownPayment(t, repo, "pay-1", "o1")

	NewReconciler(svc, time.Minute, 10, nil).RunOnce(t.Context())

	stored, err := repo.GetByID(t.Context(), "pay-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusUnknown, stored.Status, "a raced refund leaves the payment for the next pass")
}
