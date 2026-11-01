// Package fake is a deterministic in-process payment provider used in place of a
// real gateway. It approves by default; a configured amount is always declined,
// another always times out before settlement, and another settles before the
// answer is lost, so the failure paths are exercisable end to end.
package fake

import (
	"context"
	"crypto/rand"
	"minimart/internal/payment/domain"
	"sync"
)

// Config selects the amounts that trigger the non-happy paths. A zero value
// disables the corresponding case.
type Config struct {
	// DeclineAmountCents is declined (a definite refusal).
	DeclineAmountCents int64
	// TimeoutAmountCents simulates a provider timeout before settlement (an
	// unknown outcome).
	TimeoutAmountCents int64
	// SettleThenTimeoutAmountCents settles the charge and then loses the answer.
	// The returned result still carries the provider reference so local state
	// can be reconciled.
	SettleThenTimeoutAmountCents int64
}

// Provider is the fake implementation of domain.Provider. It keeps a small
// in-memory ledger keyed by the payment intent's idempotency key, so repeated
// attempts reuse the same provider reference instead of charging twice.
type Provider struct {
	cfg Config

	mu         sync.Mutex
	references map[string]string
	settled    map[string]bool
	authorizes map[string]int
	captures   map[string]int
}

var _ domain.Provider = (*Provider)(nil)

// New builds a fake provider from cfg.
func New(cfg Config) *Provider {
	return &Provider{
		cfg:        cfg,
		references: make(map[string]string),
		settled:    make(map[string]bool),
		authorizes: make(map[string]int),
		captures:   make(map[string]int),
	}
}

// Name returns the provider identifier persisted with each payment.
func (p *Provider) Name() string { return "fake" }

// Authorize approves the charge unless the amount is configured to decline or
// time out. The idempotency key, when present, pins one provider reference to
// one payment intent.
func (p *Provider) Authorize(_ context.Context, req domain.AuthorizationRequest) (domain.Result, error) {
	p.mu.Lock()
	p.authorizes[req.IdempotencyKey]++
	p.mu.Unlock()

	switch {
	case p.cfg.DeclineAmountCents != 0 && req.AmountCents == p.cfg.DeclineAmountCents:
		return domain.Result{Outcome: domain.OutcomeDeclined, Reason: "declined by fake provider"}, nil
	case p.cfg.TimeoutAmountCents != 0 && req.AmountCents == p.cfg.TimeoutAmountCents:
		return domain.Result{Outcome: domain.OutcomeTimeout, Reason: "fake provider timed out"}, nil
	default:
		return domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: p.referenceFor(req.IdempotencyKey)}, nil
	}
}

// Capture settles the authorization. For the settle-then-timeout amount the
// fake records the settlement and then reports an unknown outcome while keeping
// the provider reference, which is what reconciliation needs.
func (p *Provider) Capture(_ context.Context, req domain.CaptureRequest) (domain.Result, error) {
	ref := p.referenceFor(req.IdempotencyKey)
	if ref == "" {
		ref = req.ProviderRef
	}
	p.mu.Lock()
	p.captures[req.IdempotencyKey]++
	p.settled[ref] = true
	p.mu.Unlock()

	if p.cfg.SettleThenTimeoutAmountCents != 0 && req.AmountCents == p.cfg.SettleThenTimeoutAmountCents {
		return domain.Result{
			Outcome:     domain.OutcomeTimeout,
			ProviderRef: ref,
			Reason:      "fake provider settled but the answer was lost",
		}, nil
	}
	return domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: ref}, nil
}

// GetStatus reports an approved status for the reference. It is used by the
// reconciliation worker to resolve unknown payments.
func (p *Provider) GetStatus(_ context.Context, providerRef string) (domain.Result, error) {
	return domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: providerRef}, nil
}

// Refund echoes the reference and approves; the fake has no settlement ledger
// to reverse.
func (p *Provider) Refund(_ context.Context, providerRef string, _ int64) (domain.Result, error) {
	return domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: providerRef}, nil
}

// AuthorizeCalls returns how many times the provider was asked to authorize a
// given intent key. It lets tests assert that retries do not create new charges.
func (p *Provider) AuthorizeCalls(idempotencyKey string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.authorizes[idempotencyKey]
}

// CaptureCalls returns how many times the provider was asked to settle a given
// intent key.
func (p *Provider) CaptureCalls(idempotencyKey string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.captures[idempotencyKey]
}

func (p *Provider) referenceFor(idempotencyKey string) string {
	if idempotencyKey == "" {
		return newRef()
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if ref, ok := p.references[idempotencyKey]; ok {
		return ref
	}
	ref := newRef()
	p.references[idempotencyKey] = ref
	return ref
}

func newRef() string {
	return "fake_" + rand.Text()
}
