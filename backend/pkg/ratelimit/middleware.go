package ratelimit

import (
	"cmp"
	"log/slog"
	"math"
	"minimart/pkg/metrics"
	"minimart/pkg/response"
	"net"
	"net/http"
	"strconv"
	"time"
)

// KeyFunc derives the rate-limit bucket key for a request.
type KeyFunc func(r *http.Request) string

// Middleware wraps next with rate limiting. Requests whose key is
// within quota pass through; others get 429 with a Retry-After hint. scope labels
// the metrics and logs. When limiter is nil the middleware is a pass-through.
//
// It fails open: if the limiter's backing store is unavailable the request is
// allowed, because rate limiting is a cache-tier concern and must not become a
// new single point of failure.
func Middleware(limiter Limiter, keyFunc KeyFunc, scope string, m metrics.Metrics, next http.Handler) http.Handler {
	if limiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result, err := limiter.Allow(r.Context(), keyFunc(r))
		if err != nil {
			slog.WarnContext(r.Context(), "rate limiter unavailable; allowing request", "scope", scope, "error", err)
			m.IncRateLimited(scope, ResultError)
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		if result.Allowed {
			m.IncRateLimited(scope, ResultAllowed)
			m.ObserveRateLimitRemaining(scope, result.Remaining)
			next.ServeHTTP(w, r)
			return
		}

		m.IncRateLimited(scope, ResultThrottled)
		m.ObserveRateLimitRemaining(scope, result.Remaining)
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(result.RetryAfter)))
		response.WriteJSON(w, http.StatusTooManyRequests, response.Fail(http.StatusTooManyRequests, "limite de requisições excedido"))
	})
}

// ByHeaderOrIP keys by the named header when the client sent it, falling back to
// the client's IP otherwise. Pass "Idempotency-Key" to bucket per idempotency key
// while still limiting header-less requests (health checks, product reads) by IP.
func ByHeaderOrIP(header string) KeyFunc {
	return func(r *http.Request) string {
		return cmp.Or(r.Header.Get(header), clientIP(r))
	}
}

// clientIP returns the request's remote host without its port, falling back to
// the raw RemoteAddr when it cannot be split.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// retryAfterSeconds rounds a retry duration up to whole seconds, never below 1,
// so the Retry-After header is always actionable.
func retryAfterSeconds(d time.Duration) int {
	secs := int(math.Ceil(d.Seconds()))
	if secs < 1 {
		return 1
	}
	return secs
}
