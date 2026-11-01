package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// platformMetrics carries the cross-cutting infrastructure collectors: rate
// limiting, Redis, and idempotency.
type platformMetrics struct {
	rateLimited        *prometheus.CounterVec
	rateLimitRemaining *prometheus.HistogramVec
	redisCommands      *prometheus.CounterVec
	redisCommandDur    *prometheus.HistogramVec
	idempotency        *prometheus.CounterVec
}

// newPlatformMetrics builds and registers the platform collectors.
func newPlatformMetrics(reg prometheus.Registerer) platformMetrics {
	m := platformMetrics{
		// Dashboard: sum by (result) (rate(minimart_ratelimit_requests_total[5m]))
		// Alert:     sum(rate(minimart_ratelimit_requests_total{result="throttled"}[5m])) > 10
		rateLimited: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "ratelimit",
			Name:      "requests_total",
			Help:      "Total number of requests seen by the rate limiter by scope and result.",
		}, []string{"scope", "result"}),

		rateLimitRemaining: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "ratelimit",
			Name:      "remaining_tokens",
			Help:      "Number of tokens remaining in the bucket after a decision.",
			Buckets:   []float64{0, 1, 2, 5, 10, 20, 50, 100},
		}, []string{"scope"}),

		redisCommands: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "redis",
			Name:      "commands_total",
			Help:      "Total number of Redis commands by command and status.",
		}, []string{"command", "status"}),

		redisCommandDur: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "redis",
			Name:      "command_duration_seconds",
			Help:      "Redis command duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"command"}),

		// Dashboard: sum by (result) (rate(minimart_idempotency_events_total[5m]))
		idempotency: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "idempotency",
			Name:      "events_total",
			Help:      "Total number of idempotency-key outcomes by scope and result (miss, replay, conflict, error).",
		}, []string{"scope", "result"}),
	}
	reg.MustRegister(m.rateLimited, m.rateLimitRemaining, m.redisCommands, m.redisCommandDur, m.idempotency)
	return m
}

func (m *platformMetrics) IncRateLimited(scope, result string) {
	m.rateLimited.WithLabelValues(scope, result).Inc()
}

func (m *platformMetrics) ObserveRateLimitRemaining(scope string, remaining int) {
	m.rateLimitRemaining.WithLabelValues(scope).Observe(float64(remaining))
}

func (m *platformMetrics) IncRedisCommand(command, status string) {
	m.redisCommands.WithLabelValues(command, status).Inc()
}

func (m *platformMetrics) ObserveRedisCommandLatency(command string, d time.Duration) {
	m.redisCommandDur.WithLabelValues(command).Observe(d.Seconds())
}

func (m *platformMetrics) IncIdempotency(scope, result string) {
	m.idempotency.WithLabelValues(scope, result).Inc()
}
