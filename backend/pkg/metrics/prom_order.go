package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// orderMetrics carries the order-domain collectors.
type orderMetrics struct {
	orderPlaced         *prometheus.CounterVec
	orderAmount         prometheus.Histogram
	orderCancelled      *prometheus.CounterVec
	orderPaymentApplied *prometheus.CounterVec
}

// newOrderMetrics builds and registers the order collectors.
func newOrderMetrics(reg prometheus.Registerer) orderMetrics {
	m := orderMetrics{
		// Dashboard: sum by (status) (rate(minimart_order_placed_total[5m]))
		// Alert:     rate(minimart_order_placed_total{status="success"}[30m]) == 0
		orderPlaced: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "order",
			Name:      "placed_total",
			Help:      "Total number of orders placed by result.",
		}, []string{"status"}),

		// Dashboard: histogram_quantile(0.50, sum(rate(minimart_order_amount_cents_bucket[5m])) by (le))
		orderAmount: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "order",
			Name:      "amount_cents",
			Help:      "Order total value in cents.",
			Buckets:   []float64{100, 500, 1000, 2500, 5000, 10000, 25000, 50000, 100000, 500000},
		}),

		// Dashboard: sum by (status) (rate(minimart_order_cancelled_total[5m]))
		orderCancelled: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "order",
			Name:      "cancelled_total",
			Help:      "Total number of orders cancelled by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_order_payment_applied_total[5m]))
		orderPaymentApplied: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "order",
			Name:      "payment_applied_total",
			Help:      "Total number of payment outcomes applied to orders by result.",
		}, []string{"status"}),
	}
	reg.MustRegister(m.orderPlaced, m.orderAmount, m.orderCancelled, m.orderPaymentApplied)
	return m
}

func (m *orderMetrics) IncOrderPlaced(status string) {
	m.orderPlaced.WithLabelValues(status).Inc()
}

func (m *orderMetrics) ObserveOrderAmount(cents float64) { m.orderAmount.Observe(cents) }

func (m *orderMetrics) IncOrderCancelled(status string) {
	m.orderCancelled.WithLabelValues(status).Inc()
}

func (m *orderMetrics) IncOrderPaymentApplied(status string) {
	m.orderPaymentApplied.WithLabelValues(status).Inc()
}
