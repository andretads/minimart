package domain

import "context"

// Outcome is the provider's classification of a charge attempt.
type Outcome int

const (
	// OutcomeApproved means the provider authorized and captured the amount.
	OutcomeApproved Outcome = iota
	// OutcomeDeclined means the provider explicitly refused the charge.
	OutcomeDeclined
	// OutcomeTechnicalError means the provider could not be reached or returned
	// an unexpected error; the charge's fate is unknown.
	OutcomeTechnicalError
	// OutcomeTimeout means the provider did not answer in time; the charge may
	// or may not have gone through.
	OutcomeTimeout
)

// Result is the provider's answer to a charge call.
type Result struct {
	Outcome     Outcome
	ProviderRef string
	Reason      string
}

// AuthorizationRequest carries everything a provider needs to authorize a charge.
type AuthorizationRequest struct {
	PaymentID   string
	OrderID     string
	AmountCents int64
	Currency    string
	Method      string
}

// Provider is the payment-gateway port. Authorize reserves funds and Capture
// settles them; GetStatus is reserved for future reconciliation of unknown
// payments. Refund reverses a captured charge. Implementations live in
// infrastructure so the application layer depends only on this interface.
type Provider interface {
	// Name identifies the provider; it is persisted with each payment.
	Name() string
	Authorize(ctx context.Context, req AuthorizationRequest) (Result, error)
	Capture(ctx context.Context, providerRef string, amountCents int64) (Result, error)
	// Refund reverses a captured charge. An approved result carries the
	// provider reference of the reversed charge; a declined result reports why.
	Refund(ctx context.Context, providerRef string, amountCents int64) (Result, error)
	GetStatus(ctx context.Context, providerRef string) (Result, error)
}
