package resilient

import (
	"context"
	"minimart/pkg/idempotency"
	"minimart/pkg/resilience"
	"time"
)

// Store wraps an idempotency.Store with a circuit breaker and a per-call
// timeout. It returns the underlying error (or resilience.ErrOpen) so callers
// can fail open.
type Store struct {
	inner   idempotency.Store
	breaker *resilience.Breaker
	timeout time.Duration
}

// New wraps inner.
func New(inner idempotency.Store, breaker *resilience.Breaker, timeout time.Duration) *Store {
	if timeout <= 0 {
		timeout = 200 * time.Millisecond
	}
	return &Store{inner: inner, breaker: breaker, timeout: timeout}
}

func (s *Store) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, s.timeout)
}

func (s *Store) Get(ctx context.Context, key string) (idempotency.Record, bool, error) {
	var rec idempotency.Record
	var found bool
	err := s.breaker.Execute(ctx, func(ctx context.Context) error {
		c, cancel := s.withTimeout(ctx)
		defer cancel()
		r, f, err := s.inner.Get(c, key)
		rec, found = r, f
		return err
	})
	return rec, found, err
}

func (s *Store) Reserve(ctx context.Context, key, fingerprint string, ttl time.Duration) (bool, error) {
	var ok bool
	err := s.breaker.Execute(ctx, func(ctx context.Context) error {
		c, cancel := s.withTimeout(ctx)
		defer cancel()
		o, err := s.inner.Reserve(c, key, fingerprint, ttl)
		ok = o
		return err
	})
	return ok, err
}

func (s *Store) Complete(ctx context.Context, key, fingerprint, resourceID string, ttl time.Duration) error {
	return s.breaker.Execute(ctx, func(ctx context.Context) error {
		c, cancel := s.withTimeout(ctx)
		defer cancel()
		return s.inner.Complete(c, key, fingerprint, resourceID, ttl)
	})
}

func (s *Store) Release(ctx context.Context, key, fingerprint string) error {
	return s.breaker.Execute(ctx, func(ctx context.Context) error {
		c, cancel := s.withTimeout(ctx)
		defer cancel()
		return s.inner.Release(c, key, fingerprint)
	})
}
