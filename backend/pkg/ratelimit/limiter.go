// Package ratelimit provides a rate limiter port and an HTTP
// middleware that rejects requests exceeding a key's quota with 429.
//
// The limiter is a cache-tier concern: when the backing store (Redis) is
// unavailable, the middleware fails open and lets the request through, because
// losing rate limiting is preferable to taking down the API.
package ratelimit

import (
	"context"
	"time"
)

// Result label values passed to metrics.IncRateLimited. Keeping them here stops
// callers from inventing spellings that would split a series.
const (
	ResultAllowed   = "allowed"
	ResultThrottled = "throttled"
	ResultError     = "error"
)

// Result is the outcome of one Allow call.
type Result struct {
	// Allowed reports whether the key is within its quota for the window.
	Allowed bool
	// Remaining is how many requests the key may still make this window.
	Remaining int
	// RetryAfter is how long until the window resets, used as the Retry-After
	// hint when a request is throttled.
	RetryAfter time.Duration
}

// Limiter decides whether a key may proceed. An error means the backing store
// was unavailable; callers fail open.
type Limiter interface {
	Allow(ctx context.Context, key string) (Result, error)
}
