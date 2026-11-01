package ratelimit

import (
	"context"
	"errors"
	"minimart/pkg/metrics"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeLimiter is a hand-written test double for the Limiter port.
type fakeLimiter struct {
	result  Result
	err     error
	lastKey string
}

func (f *fakeLimiter) Allow(_ context.Context, key string) (Result, error) {
	f.lastKey = key
	return f.result, f.err
}

// countingMetrics records rate-limit outcomes while no-op'ing every other signal.
type countingMetrics struct {
	metrics.Noop
	rateLimited [][2]string
}

func (c *countingMetrics) IncRateLimited(scope, result string) {
	c.rateLimited = append(c.rateLimited, [2]string{scope, result})
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestMiddlewareAllowed(t *testing.T) {
	lim := &fakeLimiter{result: Result{Allowed: true, Remaining: 7}}
	m := &countingMetrics{}
	h := Middleware(lim, func(*http.Request) string { return "k" }, "global", m, okHandler())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "7", rec.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, [][2]string{{"global", ResultAllowed}}, m.rateLimited)
}

func TestMiddlewareThrottled(t *testing.T) {
	lim := &fakeLimiter{result: Result{Allowed: false, Remaining: 0, RetryAfter: 1500 * time.Millisecond}}
	m := &countingMetrics{}
	h := Middleware(lim, func(*http.Request) string { return "k" }, "global", m, okHandler())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "2", rec.Header().Get("Retry-After"))
	assert.Equal(t, [][2]string{{"global", ResultThrottled}}, m.rateLimited)
}

func TestMiddlewareFailsOpenOnLimiterError(t *testing.T) {
	lim := &fakeLimiter{err: errors.New("redis down")}
	m := &countingMetrics{}
	h := Middleware(lim, func(*http.Request) string { return "k" }, "global", m, okHandler())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, [][2]string{{"global", ResultError}}, m.rateLimited)
}

func TestMiddlewareNilLimiterPassesThrough(t *testing.T) {
	m := &countingMetrics{}
	h := Middleware(nil, func(*http.Request) string { return "k" }, "global", m, okHandler())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, m.rateLimited)
}

func TestByHeaderOrIP(t *testing.T) {
	t.Run("uses the header when present", func(t *testing.T) {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		r.Header.Set("Idempotency-Key", "abc")
		assert.Equal(t, "abc", ByHeaderOrIP("Idempotency-Key")(r))
	})

	t.Run("falls back to the client IP when the header is absent", func(t *testing.T) {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		r.RemoteAddr = "192.0.2.7:54321"
		assert.Equal(t, "192.0.2.7", ByHeaderOrIP("Idempotency-Key")(r))
	})

	t.Run("returns the raw address when it has no port", func(t *testing.T) {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		r.RemoteAddr = "192.0.2.7"
		assert.Equal(t, "192.0.2.7", ByHeaderOrIP("Idempotency-Key")(r))
	})
}

func TestRetryAfterSeconds(t *testing.T) {
	assert.Equal(t, 1, retryAfterSeconds(0))
	assert.Equal(t, 1, retryAfterSeconds(400*time.Millisecond))
	assert.Equal(t, 2, retryAfterSeconds(1500*time.Millisecond))
	assert.Equal(t, 5, retryAfterSeconds(5*time.Second))
}

func TestFakeLimiterRecordsKey(t *testing.T) {
	lim := &fakeLimiter{result: Result{Allowed: true}}
	_, err := lim.Allow(t.Context(), "the-key")
	require.NoError(t, err)
	assert.Equal(t, "the-key", lim.lastKey)
}
