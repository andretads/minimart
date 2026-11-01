package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPayment(t *testing.T) Payment {
	t.Helper()
	p, err := NewPayment("pay-1", "ord-1", "user-1", "idem-1", 1500, "USD", "card", "fake")
	require.NoError(t, err)
	return p
}

func TestNewPayment(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		amount   int64
		currency string
		method   string
		wantErr  error
	}{
		{"valid", "user-1", 1500, "USD", "card", nil},
		{"missing user", "", 1500, "USD", "card", ErrPaymentUserRequired},
		{"zero amount", "user-1", 0, "USD", "card", ErrPaymentAmountInvalid},
		{"negative amount", "user-1", -1, "USD", "card", ErrPaymentAmountInvalid},
		{"missing currency", "user-1", 1500, "", "card", ErrPaymentCurrencyRequired},
		{"missing method", "user-1", 1500, "USD", "", ErrPaymentMethodRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewPayment("pay-1", "ord-1", tt.userID, "idem-1", tt.amount, tt.currency, tt.method, "fake")
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, StatusPending, p.Status)
			assert.Equal(t, "user-1", p.UserID)
			assert.Equal(t, "fake", p.Provider)
			assert.Nil(t, p.ProviderRef)
			require.Len(t, p.Events(), 1)
			_, ok := p.Events()[0].(PaymentInitiatedEvent)
			assert.True(t, ok, "creation raises PaymentInitiatedEvent")
		})
	}
}

func TestApplyOutcome(t *testing.T) {
	tests := []struct {
		name       string
		outcome    Outcome
		wantStatus Status
		wantEvent  func(Event) bool
	}{
		{
			name:       "approved succeeds",
			outcome:    OutcomeApproved,
			wantStatus: StatusSucceeded,
			wantEvent:  func(e Event) bool { _, ok := e.(PaymentSucceededEvent); return ok },
		},
		{
			name:       "declined fails",
			outcome:    OutcomeDeclined,
			wantStatus: StatusFailed,
			wantEvent:  func(e Event) bool { _, ok := e.(PaymentFailedEvent); return ok },
		},
		{
			name:       "technical error is unknown, not failed",
			outcome:    OutcomeTechnicalError,
			wantStatus: StatusUnknown,
			wantEvent:  func(e Event) bool { _, ok := e.(PaymentUnknownEvent); return ok },
		},
		{
			name:       "timeout is unknown, not failed",
			outcome:    OutcomeTimeout,
			wantStatus: StatusUnknown,
			wantEvent:  func(e Event) bool { _, ok := e.(PaymentUnknownEvent); return ok },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestPayment(t)
			before := p.UpdatedAt

			got, err := p.ApplyOutcome(Result{Outcome: tt.outcome, ProviderRef: "ref-1", Reason: "because"})
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, got.Status)
			assert.Equal(t, StatusPending, p.Status, "ApplyOutcome does not mutate the receiver")
			assert.False(t, got.UpdatedAt.Before(before))

			// The outcome event is appended after the initiated event.
			require.Len(t, got.Events(), 2)
			assert.True(t, tt.wantEvent(got.Events()[1]))
		})
	}
}

func TestApplyOutcomeRejectsNonPending(t *testing.T) {
	p := newTestPayment(t)
	approved, err := p.ApplyOutcome(Result{Outcome: OutcomeApproved})
	require.NoError(t, err)

	_, err = approved.ApplyOutcome(Result{Outcome: OutcomeApproved})
	assert.ErrorIs(t, err, ErrPaymentNotPending)
}

func TestApplyOutcomeRejectsUnknownOutcome(t *testing.T) {
	p := newTestPayment(t)
	_, err := p.ApplyOutcome(Result{Outcome: Outcome(42)})
	assert.ErrorIs(t, err, ErrInvalidTransition)
}

func TestApplyOutcomeSetsProviderRef(t *testing.T) {
	p := newTestPayment(t)
	got, err := p.ApplyOutcome(Result{Outcome: OutcomeApproved, ProviderRef: "charge-xyz"})
	require.NoError(t, err)
	require.NotNil(t, got.ProviderRef)
	assert.Equal(t, "charge-xyz", *got.ProviderRef)

	// A blank provider ref leaves ProviderRef nil rather than pointing at "".
	p2 := newTestPayment(t)
	got2, err := p2.ApplyOutcome(Result{Outcome: OutcomeDeclined, Reason: "no funds"})
	require.NoError(t, err)
	assert.Nil(t, got2.ProviderRef)
	require.NotNil(t, got2.FailureReason)
	assert.Equal(t, "no funds", *got2.FailureReason)
}

func TestNewPaymentStampsTimes(t *testing.T) {
	before := time.Now().UTC()
	p := newTestPayment(t)
	assert.False(t, p.CreatedAt.Before(before))
	assert.False(t, p.CreatedAt.After(time.Now().UTC()))
	assert.Equal(t, p.CreatedAt, p.UpdatedAt)
}

func TestRefund(t *testing.T) {
	tests := []struct {
		name       string
		fromStatus Status
	}{
		{"pending charge in flight", StatusPending},
		{"succeeded", StatusSucceeded},
		{"unknown reconciled as captured", StatusUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := paymentIn(t, tt.fromStatus)
			before := p.UpdatedAt

			got, err := p.Refund(Result{Outcome: OutcomeApproved, ProviderRef: "ref-1", Reason: "pedido cancelado"})
			require.NoError(t, err)

			assert.Equal(t, StatusRefunded, got.Status)
			assert.Equal(t, tt.fromStatus, p.Status, "Refund does not mutate the receiver")
			assert.False(t, got.UpdatedAt.Before(before))
			require.NotNil(t, got.ProviderRef)
			assert.Equal(t, "ref-1", *got.ProviderRef)
			assert.Nil(t, got.FailureReason, "the refund supersedes the unknown outcome's reason")

			require.Len(t, got.Events(), len(p.Events())+1)
			ev, ok := got.Events()[len(got.Events())-1].(PaymentRefundedEvent)
			assert.True(t, ok, "refund raises PaymentRefundedEvent")
			assert.Equal(t, got.ID, ev.PaymentID)
			assert.Equal(t, "pedido cancelado", ev.Reason)
		})
	}
}

func TestRefundRejectsNonRefundable(t *testing.T) {
	for _, status := range []Status{StatusFailed, StatusRefunded} {
		t.Run(string(status), func(t *testing.T) {
			p := paymentIn(t, status)
			_, err := p.Refund(Result{Outcome: OutcomeApproved, ProviderRef: "ref-1"})
			assert.ErrorIs(t, err, ErrPaymentNotRefundable)
		})
	}
}

func TestRefundRejectsNotApproved(t *testing.T) {
	for _, outcome := range []Outcome{OutcomeDeclined, OutcomeTechnicalError, OutcomeTimeout} {
		t.Run(outcomeName(outcome), func(t *testing.T) {
			p := paymentIn(t, StatusSucceeded)
			_, err := p.Refund(Result{Outcome: outcome, Reason: "no"})
			assert.ErrorIs(t, err, ErrInvalidTransition)
		})
	}
}

func TestRefundFallsBackToPaymentProviderRef(t *testing.T) {
	p := paymentIn(t, StatusSucceeded)
	got, err := p.Refund(Result{Outcome: OutcomeApproved})
	require.NoError(t, err)
	require.NotNil(t, got.ProviderRef)
	assert.Equal(t, "ref-1", *got.ProviderRef, "an empty result ref reuses the payment's charge ref")
}

func TestRefundClearsUnknownFailureReason(t *testing.T) {
	p := paymentIn(t, StatusUnknown)
	require.NotNil(t, p.FailureReason)

	got, err := p.Refund(Result{Outcome: OutcomeApproved, ProviderRef: "ref-1"})
	require.NoError(t, err)
	assert.Nil(t, got.FailureReason)
}

// paymentIn returns a payment already in the given status, carrying a charge
// reference and, for unknown, a failure reason.
func paymentIn(t *testing.T, status Status) Payment {
	t.Helper()
	p, err := NewPayment("pay-1", "ord-1", "user-1", "idem-1", 1500, "USD", "card", "fake")
	require.NoError(t, err)
	switch status {
	case StatusPending:
		return p
	case StatusSucceeded:
		p, err = p.ApplyOutcome(Result{Outcome: OutcomeApproved, ProviderRef: "ref-1"})
	case StatusUnknown:
		p, err = p.ApplyOutcome(Result{Outcome: OutcomeTimeout, Reason: "slow"})
	case StatusFailed:
		p, err = p.ApplyOutcome(Result{Outcome: OutcomeDeclined, Reason: "no funds"})
	case StatusRefunded:
		p, err = p.ApplyOutcome(Result{Outcome: OutcomeApproved, ProviderRef: "ref-1"})
		require.NoError(t, err)
		p, err = p.Refund(Result{Outcome: OutcomeApproved, ProviderRef: "ref-1"})
	}
	require.NoError(t, err)
	return p
}

func outcomeName(o Outcome) string {
	switch o {
	case OutcomeApproved:
		return "approved"
	case OutcomeDeclined:
		return "declined"
	case OutcomeTechnicalError:
		return "technical error"
	case OutcomeTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}
