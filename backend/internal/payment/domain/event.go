package domain

// Event is a domain event raised by the Payment aggregate. Infrastructure maps
// each concrete event to a serialized broker payload in the same transaction that
// persists the payment, so events are published at least once via the outbox.
type Event interface {
	// isPaymentEvent seals the interface to this package's event types.
	isPaymentEvent()
}

// PaymentInitiatedEvent records that a payment was created and is pending.
type PaymentInitiatedEvent struct {
	PaymentID   string
	OrderID     string
	UserID      string
	AmountCents int64
	Currency    string
}

func (PaymentInitiatedEvent) isPaymentEvent() {}

// PaymentSucceededEvent records that a payment was captured successfully.
type PaymentSucceededEvent struct {
	PaymentID   string
	OrderID     string
	UserID      string
	AmountCents int64
	Currency    string
	ProviderRef string
}

func (PaymentSucceededEvent) isPaymentEvent() {}

// PaymentFailedEvent records that the provider declined a payment.
type PaymentFailedEvent struct {
	PaymentID   string
	OrderID     string
	UserID      string
	AmountCents int64
	Currency    string
	Reason      string
}

func (PaymentFailedEvent) isPaymentEvent() {}

// PaymentUnknownEvent records that a payment's outcome could not be determined
// and must be reconciled later.
type PaymentUnknownEvent struct {
	PaymentID   string
	OrderID     string
	UserID      string
	AmountCents int64
	Currency    string
	Reason      string
}

func (PaymentUnknownEvent) isPaymentEvent() {}

// PaymentRefundedEvent records that a captured payment's charge was reversed
// with the provider, either because the order was cancelled while the charge
// was in flight or after the payment had already succeeded.
type PaymentRefundedEvent struct {
	PaymentID   string
	OrderID     string
	UserID      string
	AmountCents int64
	Currency    string
	ProviderRef string
	Reason      string
}

func (PaymentRefundedEvent) isPaymentEvent() {}
