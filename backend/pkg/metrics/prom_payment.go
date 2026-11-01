package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// paymentMetrics carries the payment-domain collectors.
type paymentMetrics struct {
	paymentEvents *prometheus.CounterVec
	paymentAmount prometheus.Histogram
}

// newPaymentMetrics builds and registers the payment collectors.
func newPaymentMetrics(reg prometheus.Registerer) paymentMetrics {
	m := paymentMetrics{
		// Dashboard: sum by (state) (rate(minimart_payment_events_total[5m]))
		// Alert:     sum(rate(minimart_payment_events_total{state="unknown"}[15m])) > 0
		paymentEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "payment",
			Name:      "events_total",
			Help:      "Total number of payments by state (initiated, succeeded, failed, unknown).",
		}, []string{"state"}),

		// Dashboard: histogram_quantile(0.50, sum(rate(minimart_payment_amount_cents_bucket[5m])) by (le))
		paymentAmount: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "payment",
			Name:      "amount_cents",
			Help:      "Payment amount in cents.",
			Buckets:   []float64{100, 500, 1000, 2500, 5000, 10000, 25000, 50000, 100000, 500000},
		}),
	}
	reg.MustRegister(m.paymentEvents, m.paymentAmount)
	return m
}

func (m *paymentMetrics) IncPayment(state string) {
	m.paymentEvents.WithLabelValues(state).Inc()
}

func (m *paymentMetrics) ObservePaymentAmount(cents float64) { m.paymentAmount.Observe(cents) }
