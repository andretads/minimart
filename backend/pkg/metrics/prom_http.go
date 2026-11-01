package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// httpMetrics carries the HTTP golden-signal collectors.
type httpMetrics struct {
	requests *prometheus.CounterVec
	latency  *prometheus.HistogramVec
	inFlight prometheus.Gauge
}

// newHTTPMetrics builds and registers the HTTP collectors.
func newHTTPMetrics(reg prometheus.Registerer) httpMetrics {
	m := httpMetrics{
		// Dashboard: sum by (status) (rate(minimart_http_requests_total[5m]))
		// Alert:     sum(rate(minimart_http_requests_total{status=~"5.."}[5m])) / sum(rate(minimart_http_requests_total[5m])) > 0.01
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests by route, method, and status.",
		}, []string{"endpoint", "method", "status"}),

		// Dashboard: histogram_quantile(0.99, sum(rate(minimart_http_request_duration_seconds_bucket[5m])) by (le, endpoint))
		// Alert:     histogram_quantile(0.99, sum(rate(minimart_http_request_duration_seconds_bucket[5m])) by (le)) > 2
		latency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"endpoint", "method"}),

		// Dashboard: minimart_http_requests_in_flight
		// Alert:     minimart_http_requests_in_flight > 500
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "Number of HTTP requests currently being served.",
		}),
	}
	reg.MustRegister(m.requests, m.latency, m.inFlight)
	return m
}

func (m *httpMetrics) IncRequest(endpoint, method, status string) {
	m.requests.WithLabelValues(endpoint, method, status).Inc()
}

func (m *httpMetrics) ObserveLatency(endpoint, method string, d time.Duration) {
	m.latency.WithLabelValues(endpoint, method).Observe(d.Seconds())
}

func (m *httpMetrics) IncInFlight() { m.inFlight.Inc() }
func (m *httpMetrics) DecInFlight() { m.inFlight.Dec() }
