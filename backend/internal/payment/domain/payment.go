package domain

import (
	"time"
)

// Status is a payment's lifecycle state.
type Status string

const (
	// StatusPending is the state of a payment whose provider call is in flight.
	StatusPending Status = "pending"
	// StatusSucceeded is a payment the provider authorized and captured.
	StatusSucceeded Status = "succeeded"
	// StatusFailed is a payment the provider explicitly declined.
	StatusFailed Status = "failed"
	// StatusUnknown is a payment whose outcome could not be determined: a
	// provider timeout or technical error. It is deliberately distinct from
	// failed because the charge may in fact have succeeded. Reconciling unknown
	// payments with the provider is a separate, deferred concern.
	StatusUnknown Status = "unknown"
	// StatusRefunded is a captured payment whose charge was reversed with the
	// provider. It is reached from pending (the order was cancelled while the
	// charge was in flight, so the capture is reversed before it is finalized)
	// or from succeeded (the order was cancelled after the payment succeeded).
	StatusRefunded Status = "refunded"
)

// Payment is a payment attempt for an order. Like Product and Order, it is valid
// by construction: NewPayment is the only way to make one, and ApplyOutcome is
// the only way to move it out of pending. Statuses are terminal except pending:
// once a payment leaves pending it is never reprocessed.
type Payment struct {
	ID             string
	OrderID        string
	UserID         string
	IdempotencyKey string
	AmountCents    int64
	Currency       string
	Method         string
	Status         Status
	Provider       string
	ProviderRef    *string
	FailureReason  *string
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// events are the domain events raised since the payment was created or
	// loaded. The repository persists them to the outbox in the same transaction
	// as the payment, then they are discarded with the value.
	events []Event
}

// Events returns the domain events raised since the payment was created or loaded.
func (p Payment) Events() []Event { return p.events }

// NewPayment validates the request's core fields, stamps the timestamps, and
// raises the PaymentInitiated event. The payment starts pending; the provider is
// called by the application layer afterwards. userID is the user placing the
// order the payment is for; it rides on every event so consumers (for example the
// notification domain) can attribute the event without a lookup.
func NewPayment(id, orderID, userID, idempotencyKey string, amountCents int64, currency, method, provider string) (Payment, error) {
	if amountCents <= 0 {
		return Payment{}, ErrPaymentAmountInvalid
	}
	if currency == "" {
		return Payment{}, ErrPaymentCurrencyRequired
	}
	if method == "" {
		return Payment{}, ErrPaymentMethodRequired
	}
	if userID == "" {
		return Payment{}, ErrPaymentUserRequired
	}
	now := time.Now().UTC()
	return Payment{
		ID:             id,
		OrderID:        orderID,
		UserID:         userID,
		IdempotencyKey: idempotencyKey,
		AmountCents:    amountCents,
		Currency:       currency,
		Method:         method,
		Status:         StatusPending,
		Provider:       provider,
		CreatedAt:      now,
		UpdatedAt:      now,
		events: []Event{PaymentInitiatedEvent{
			PaymentID:   id,
			OrderID:     orderID,
			UserID:      userID,
			AmountCents: amountCents,
			Currency:    currency,
		}},
	}, nil
}

// ApplyOutcome returns a copy of the payment transitioned by the provider's
// result, raising the matching event. Applying an outcome to a payment that is
// no longer pending is a caller error, reported as ErrPaymentNotPending.
func (p Payment) ApplyOutcome(r Result) (Payment, error) {
	if p.Status != StatusPending {
		return Payment{}, ErrPaymentNotPending
	}
	now := time.Now().UTC()
	p.UpdatedAt = now
	if r.ProviderRef != "" {
		p.ProviderRef = &r.ProviderRef
	}

	switch r.Outcome {
	case OutcomeApproved:
		p.Status = StatusSucceeded
		p.events = append(p.events, PaymentSucceededEvent{
			PaymentID:   p.ID,
			OrderID:     p.OrderID,
			UserID:      p.UserID,
			AmountCents: p.AmountCents,
			Currency:    p.Currency,
			ProviderRef: r.ProviderRef,
		})
	case OutcomeDeclined:
		p.Status = StatusFailed
		p.FailureReason = &r.Reason
		p.events = append(p.events, PaymentFailedEvent{
			PaymentID:   p.ID,
			OrderID:     p.OrderID,
			UserID:      p.UserID,
			AmountCents: p.AmountCents,
			Currency:    p.Currency,
			Reason:      r.Reason,
		})
	case OutcomeTechnicalError, OutcomeTimeout:
		// A timeout or technical error must NOT be recorded as failed: the
		// charge may have gone through. Record unknown and defer reconciliation.
		p.Status = StatusUnknown
		p.FailureReason = &r.Reason
		p.events = append(p.events, PaymentUnknownEvent{
			PaymentID:   p.ID,
			OrderID:     p.OrderID,
			UserID:      p.UserID,
			AmountCents: p.AmountCents,
			Currency:    p.Currency,
			Reason:      r.Reason,
		})
	default:
		return Payment{}, ErrInvalidTransition
	}
	return p, nil
}

// Refund returns a copy of the payment transitioned to refunded after the
// provider approved reversing its charge, raising the matching event. It is
// valid for a pending payment that was captured but not yet finalized (the
// order was cancelled while the charge was in flight), for a succeeded
// payment, and for an unknown payment the caller confirmed as captured with
// the provider: the approved refund result is the evidence that a charge
// existed to reverse. Any other state is a caller error, reported as
// ErrPaymentNotRefundable. A refund result that is not approved is reported
// as ErrInvalidTransition.
func (p Payment) Refund(r Result) (Payment, error) {
	switch p.Status {
	case StatusPending, StatusSucceeded, StatusUnknown:
	default:
		return Payment{}, ErrPaymentNotRefundable
	}
	if r.Outcome != OutcomeApproved {
		return Payment{}, ErrInvalidTransition
	}

	providerRef := r.ProviderRef
	if providerRef == "" && p.ProviderRef != nil {
		providerRef = *p.ProviderRef
	}

	now := time.Now().UTC()
	p.UpdatedAt = now
	if providerRef != "" {
		p.ProviderRef = &providerRef
	}
	p.Status = StatusRefunded
	p.FailureReason = nil
	p.events = append(p.events, PaymentRefundedEvent{
		PaymentID:   p.ID,
		OrderID:     p.OrderID,
		UserID:      p.UserID,
		AmountCents: p.AmountCents,
		Currency:    p.Currency,
		ProviderRef: providerRef,
		Reason:      r.Reason,
	})
	return p, nil
}

// ApplyReconciliation returns a copy of the payment transitioned by a provider
// status lookup. It is only valid for pending or unknown payments; terminal
// payments are never reprocessed.
func (p Payment) ApplyReconciliation(r Result) (Payment, error) {
	if p.Status != StatusPending && p.Status != StatusUnknown {
		return Payment{}, ErrPaymentNotPending
	}
	now := time.Now().UTC()
	switch r.Outcome {
	case OutcomeApproved:
		p.UpdatedAt = now
		if r.ProviderRef != "" {
			p.ProviderRef = &r.ProviderRef
		}
		p.Status = StatusSucceeded
		p.events = append(p.events, PaymentSucceededEvent{
			PaymentID:   p.ID,
			OrderID:     p.OrderID,
			UserID:      p.UserID,
			AmountCents: p.AmountCents,
			Currency:    p.Currency,
			ProviderRef: r.ProviderRef,
		})
	case OutcomeDeclined:
		p.UpdatedAt = now
		if r.ProviderRef != "" {
			p.ProviderRef = &r.ProviderRef
		}
		p.Status = StatusFailed
		p.FailureReason = &r.Reason
		p.events = append(p.events, PaymentFailedEvent{
			PaymentID:   p.ID,
			OrderID:     p.OrderID,
			UserID:      p.UserID,
			AmountCents: p.AmountCents,
			Currency:    p.Currency,
			Reason:      r.Reason,
		})
	case OutcomeTechnicalError, OutcomeTimeout:
		if p.Status == StatusUnknown {
			return p, nil
		}
		p.UpdatedAt = now
		if r.ProviderRef != "" {
			p.ProviderRef = &r.ProviderRef
		}
		p.Status = StatusUnknown
		p.FailureReason = &r.Reason
		p.events = append(p.events, PaymentUnknownEvent{
			PaymentID:   p.ID,
			OrderID:     p.OrderID,
			UserID:      p.UserID,
			AmountCents: p.AmountCents,
			Currency:    p.Currency,
			Reason:      r.Reason,
		})
	default:
		return Payment{}, ErrInvalidTransition
	}
	return p, nil
}
