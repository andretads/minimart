package resilient

import (
	"context"
	"minimart/pkg/ratelimit"
	"minimart/pkg/resilience"
	"time"
)

// Limiter wraps a ratelimit.Limiter with a circuit breaker and timeout.
type Limiter struct {
	inner   ratelimit.Limiter
	breaker *resilience.Breaker
	timeout time.Duration
}

// New wraps inner.
func New(inner ratelimit.Limiter, breaker *resilience.Breaker, timeout time.Duration) *Limiter {
	if timeout <= 0 {
		timeout = 200 * time.Millisecond
	}
	return &Limiter{inner: inner, breaker: breaker, timeout: timeout}
}

// Allow delegates to the wrapped limiter.
func (l *Limiter) Allow(ctx context.Context, key string) (ratelimit.Result, error) {
	var res ratelimit.Result
	err := l.breaker.Execute(ctx, func(ctx context.Context) error {
		c, cancel := context.WithTimeout(ctx, l.timeout)
		defer cancel()
		r, err := l.inner.Allow(c, key)
		res = r
		return err
	})
	return res, err
}
