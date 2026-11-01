// Package fake is a deterministic in-process payment provider used in place of a
// real gateway. It approves by default; a configured amount is always declined
// and another always times out, so the failure paths are exercisable end to end.
package fake

import (
	"context"
	"crypto/rand"
	"minimart/internal/payment/domain"
)

// Config selects the amounts that trigger the non-happy paths. A zero value
// disables the corresponding case.
type Config struct {
	// DeclineAmountCents is declined (a definite refusal).
	DeclineAmountCents int64
	// TimeoutAmountCents simulates a provider timeout (an unknown outcome).
	TimeoutAmountCents int64
}

// Provider is the fake implementation of domain.Provider.
type Provider struct {
	cfg Config
}

var _ domain.Provider = (*Provider)(nil)

// New builds a fake provider from cfg.
func New(cfg Config) *Provider {
	return &Provider{cfg: cfg}
}

// Name returns the provider identifier persisted with each payment.
func (p *Provider) Name() string { return "fake" }

// Authorize approves the charge unless the amount is configured to decline or
// time out.
func (p *Provider) Authorize(_ context.Context, req domain.AuthorizationRequest) (domain.Result, error) {
	switch {
	case p.cfg.DeclineAmountCents != 0 && req.AmountCents == p.cfg.DeclineAmountCents:
		return domain.Result{Outcome: domain.OutcomeDeclined, Reason: "declined by fake provider"}, nil
	case p.cfg.TimeoutAmountCents != 0 && req.AmountCents == p.cfg.TimeoutAmountCents:
		return domain.Result{Outcome: domain.OutcomeTimeout, Reason: "fake provider timed out"}, nil
	default:
		return domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: newRef()}, nil
	}
}

// Capture echoes the authorization's reference and approves; the fake has no
// separate settlement step.
func (p *Provider) Capture(_ context.Context, providerRef string, _ int64) (domain.Result, error) {
	return domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: providerRef}, nil
}

// GetStatus reports an approved status for the reference. It exists for future
// reconciliation of unknown payments; nothing calls it yet.
func (p *Provider) GetStatus(_ context.Context, providerRef string) (domain.Result, error) {
	return domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: providerRef}, nil
}

// Refund echoes the reference and approves; the fake has no settlement ledger
// to reverse.
func (p *Provider) Refund(_ context.Context, providerRef string, _ int64) (domain.Result, error) {
	return domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: providerRef}, nil
}

func newRef() string {
	return "fake_" + rand.Text()
}
