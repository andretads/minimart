package application

import (
	"context"
	"errors"
	"minimart/internal/order/presentation/module/client"
	"minimart/internal/payment/domain"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cancelledOrderReader answers a cancelled order owned by user-1.
func cancelledOrderReader() *stubOrderReader {
	return &stubOrderReader{order: client.Order{
		ID:         "o1",
		UserID:     "u1",
		Status:     "cancelled",
		TotalCents: 1500,
		Currency:   "USD",
	}}
}

// succeededPayment stores and returns a captured payment for the given order.
func succeededPayment(t *testing.T, repo *fakeRepo, id, orderID string) domain.Payment {
	t.Helper()
	p, err := domain.NewPayment(id, orderID, "u1", "idem-"+id, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	p, err = p.ApplyOutcome(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-" + id})
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))
	return p
}

func TestRefundPayment(t *testing.T) {
	repo, idem, provider, m := newFakeRepo(), newFakeIdem(), approvedProvider(), &countingMetrics{}
	svc := NewService(repo, cancelledOrderReader(), idem, provider, m, time.Hour)
	p := succeededPayment(t, repo, "pay-1", "o1")

	got, err := svc.RefundPayment(t.Context(), "pay-1")
	require.NoError(t, err)

	assert.Equal(t, domain.StatusRefunded, got.Status)
	assert.Equal(t, 1, provider.refunded)
	stored, err := repo.GetByID(t.Context(), "pay-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusRefunded, stored.Status)

	require.Len(t, got.Events(), len(p.Events())+1)
	_, ok := got.Events()[len(got.Events())-1].(domain.PaymentRefundedEvent)
	assert.True(t, ok, "refunding appends the PaymentRefunded event")
	assert.Equal(t, []string{"refunded"}, m.payments)
}

func TestRefundPaymentIsIdempotent(t *testing.T) {
	repo, idem, provider := newFakeRepo(), newFakeIdem(), approvedProvider()
	svc := NewService(repo, cancelledOrderReader(), idem, provider, metrics.Noop{}, time.Hour)
	p := succeededPayment(t, repo, "pay-1", "o1")
	refunded, err := p.Refund(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-pay-1"})
	require.NoError(t, err)
	repo.byID["pay-1"] = refunded

	got, err := svc.RefundPayment(t.Context(), "pay-1")
	require.NoError(t, err)

	assert.Equal(t, domain.StatusRefunded, got.Status)
	assert.Equal(t, 0, provider.refunded, "an already-refunded payment is returned without touching the provider")
}

func TestRefundPaymentRejectsNonSucceeded(t *testing.T) {
	for _, status := range []domain.Status{domain.StatusPending, domain.StatusFailed, domain.StatusUnknown} {
		t.Run(string(status), func(t *testing.T) {
			repo, idem, provider := newFakeRepo(), newFakeIdem(), approvedProvider()
			svc := NewService(repo, cancelledOrderReader(), idem, provider, metrics.Noop{}, time.Hour)
			p := paymentWithStatus(t, status)
			require.NoError(t, repo.Save(t.Context(), p))

			_, err := svc.RefundPayment(t.Context(), p.ID)
			require.ErrorIs(t, err, domain.ErrPaymentNotRefundable)
			assert.Equal(t, 0, provider.refunded)
		})
	}
}

func TestRefundPaymentRequiresCancelledOrder(t *testing.T) {
	repo, idem, provider := newFakeRepo(), newFakeIdem(), approvedProvider()
	reader := &stubOrderReader{order: client.Order{ID: "o1", UserID: "u1", Status: "paid"}}
	svc := NewService(repo, reader, idem, provider, metrics.Noop{}, time.Hour)
	succeededPayment(t, repo, "pay-1", "o1")

	_, err := svc.RefundPayment(t.Context(), "pay-1")
	require.ErrorIs(t, err, ErrOrderNotCancelled)
	assert.Equal(t, 0, provider.refunded)
}

func TestRefundPaymentMapsProviderFailures(t *testing.T) {
	tests := []struct {
		name      string
		provider  *stubProvider
		wantError error
	}{
		{"provider error", &stubProvider{refundErr: errors.New("gateway down")}, ErrRefundFailed},
		{"provider declined", &stubProvider{refund: domain.Result{Outcome: domain.OutcomeDeclined, Reason: "no"}}, ErrRefundFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, idem := newFakeRepo(), newFakeIdem()
			svc := NewService(repo, cancelledOrderReader(), idem, tt.provider, metrics.Noop{}, time.Hour)
			succeededPayment(t, repo, "pay-1", "o1")

			_, err := svc.RefundPayment(t.Context(), "pay-1")
			require.ErrorIs(t, err, tt.wantError)

			stored, err := repo.GetByID(t.Context(), "pay-1")
			require.NoError(t, err)
			assert.Equal(t, domain.StatusSucceeded, stored.Status, "a failed refund leaves the payment untouched")
		})
	}
}

// flippingOrderReader answers placed for the pre-charge validation and
// cancelled afterwards, simulating an order cancelled while the charge was in
// flight.
type flippingOrderReader struct{ calls int }

func (f *flippingOrderReader) GetOrder(_ context.Context, _ string) (client.Order, error) {
	f.calls++
	status := "placed"
	if f.calls > 1 {
		status = "cancelled"
	}
	return client.Order{ID: "ord-1", UserID: "user-1", Status: status, TotalCents: 1500, Currency: "USD"}, nil
}

func TestCreatePaymentRefundsWhenOrderCancelledDuringCharge(t *testing.T) {
	repo, idem, provider, m := newFakeRepo(), newFakeIdem(), approvedProvider(), &countingMetrics{}
	svc := NewService(repo, &flippingOrderReader{}, idem, provider, m, time.Hour)

	got, err := svc.CreatePayment(t.Context(), testRequest())
	require.NoError(t, err)

	assert.Equal(t, domain.StatusRefunded, got.Status)
	assert.Equal(t, 1, provider.captured)
	assert.Equal(t, 1, provider.refunded, "an approved charge on a cancelled order is reversed inline")
	stored, err := repo.GetByID(t.Context(), got.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusRefunded, stored.Status)
	assert.Equal(t, []string{"refunded"}, m.payments)
	assert.Equal(t, []float64{1500}, m.amounts)
}

func TestCreatePaymentRefundFailureKeepsChargeReference(t *testing.T) {
	repo, idem := newFakeRepo(), newFakeIdem()
	provider := &stubProvider{
		auth:      domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"},
		capture:   domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"},
		refundErr: errors.New("gateway down"),
	}
	svc := NewService(repo, &flippingOrderReader{}, idem, provider, metrics.Noop{}, time.Hour)

	_, err := svc.CreatePayment(t.Context(), testRequest())
	require.ErrorIs(t, err, ErrRefundFailed)

	// The payment stays pending but keeps the charge reference, so the
	// reconciler can retry the refund instead of the money being stranded.
	stored, err := repo.GetByIdempotencyKey(t.Context(), "idem-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPending, stored.Status)
	require.NotNil(t, stored.ProviderRef)
	assert.Equal(t, "ref-1", *stored.ProviderRef)
}

func TestCreatePaymentRefundFailureIsRecoveredByReconciler(t *testing.T) {
	repo, idem := newFakeRepo(), newFakeIdem()
	retry := &stubProvider{
		auth:      domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"},
		capture:   domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"},
		refundErr: errors.New("gateway down"),
	}
	reader := &flippingOrderReader{}
	svc := NewService(repo, reader, idem, retry, metrics.Noop{}, time.Hour)

	_, err := svc.CreatePayment(t.Context(), testRequest())
	require.ErrorIs(t, err, ErrRefundFailed)

	// The gateway recovers; the reconciler finishes the refund.
	retry.refundErr = nil
	NewReconciler(svc, time.Minute, 10, nil).RunOnce(t.Context())

	stored, err := repo.GetByIdempotencyKey(t.Context(), "idem-1")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusRefunded, stored.Status)
}

func TestCreatePaymentProceedsWhenOrderStaysPayable(t *testing.T) {
	repo, idem, provider := newFakeRepo(), newFakeIdem(), approvedProvider()
	reader := &stubOrderReader{order: client.Order{ID: "ord-1", UserID: "user-1", Status: "placed", TotalCents: 1500, Currency: "USD"}}
	svc := NewService(repo, reader, idem, provider, metrics.Noop{}, time.Hour)

	got, err := svc.CreatePayment(t.Context(), testRequest())
	require.NoError(t, err)

	assert.Equal(t, domain.StatusSucceeded, got.Status)
	assert.Equal(t, 0, provider.refunded, "a payable order never triggers a refund")
}

// paymentWithStatus builds a stored-shape payment in the given status.
func paymentWithStatus(t *testing.T, status domain.Status) domain.Payment {
	t.Helper()
	p, err := domain.NewPayment("pay-status", "o1", "u1", "idem-status", 1500, "USD", "card", "fake")
	require.NoError(t, err)
	switch status {
	case domain.StatusPending:
		return p
	case domain.StatusSucceeded:
		p, err = p.ApplyOutcome(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-status"})
	case domain.StatusFailed:
		p, err = p.ApplyOutcome(domain.Result{Outcome: domain.OutcomeDeclined, Reason: "no funds"})
	case domain.StatusUnknown:
		p, err = p.ApplyOutcome(domain.Result{Outcome: domain.OutcomeTimeout, Reason: "slow", ProviderRef: "ref-status"})
	}
	require.NoError(t, err)
	return p
}
