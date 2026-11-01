package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// cartMetrics carries the cart-domain collectors.
type cartMetrics struct {
	cartCreated     *prometheus.CounterVec
	cartRetrieved   *prometheus.CounterVec
	cartItemAdded   *prometheus.CounterVec
	cartItemUpdated *prometheus.CounterVec
	cartItemRemoved *prometheus.CounterVec
	cartCleared     *prometheus.CounterVec
	cartCheckout    *prometheus.CounterVec
	cartConsumed    *prometheus.CounterVec
}

// newCartMetrics builds and registers the cart collectors.
func newCartMetrics(reg prometheus.Registerer) cartMetrics {
	m := cartMetrics{
		// Dashboard: sum by (status) (rate(minimart_cart_created_total[5m]))
		cartCreated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cart",
			Name:      "created_total",
			Help:      "Total number of carts created by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_cart_retrieved_total[5m]))
		cartRetrieved: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cart",
			Name:      "retrieved_total",
			Help:      "Total number of cart retrievals by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_cart_item_added_total[5m]))
		cartItemAdded: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cart",
			Name:      "item_added_total",
			Help:      "Total number of cart item additions by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_cart_item_updated_total[5m]))
		cartItemUpdated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cart",
			Name:      "item_updated_total",
			Help:      "Total number of cart item updates by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_cart_item_removed_total[5m]))
		cartItemRemoved: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cart",
			Name:      "item_removed_total",
			Help:      "Total number of cart item removals by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_cart_cleared_total[5m]))
		cartCleared: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cart",
			Name:      "cleared_total",
			Help:      "Total number of cart clears by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_cart_checkout_total[5m]))
		cartCheckout: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cart",
			Name:      "checkout_total",
			Help:      "Total number of cart checkouts by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_cart_consumed_total[5m]))
		cartConsumed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cart",
			Name:      "consumed_total",
			Help:      "Total number of cart consumptions by result.",
		}, []string{"status"}),
	}
	reg.MustRegister(
		m.cartCreated,
		m.cartRetrieved,
		m.cartItemAdded,
		m.cartItemUpdated,
		m.cartItemRemoved,
		m.cartCleared,
		m.cartCheckout,
		m.cartConsumed,
	)
	return m
}

func (m *cartMetrics) IncCartCreated(status string) {
	m.cartCreated.WithLabelValues(status).Inc()
}

func (m *cartMetrics) IncCartRetrieved(status string) {
	m.cartRetrieved.WithLabelValues(status).Inc()
}

func (m *cartMetrics) IncCartItemAdded(status string) {
	m.cartItemAdded.WithLabelValues(status).Inc()
}

func (m *cartMetrics) IncCartItemUpdated(status string) {
	m.cartItemUpdated.WithLabelValues(status).Inc()
}

func (m *cartMetrics) IncCartItemRemoved(status string) {
	m.cartItemRemoved.WithLabelValues(status).Inc()
}

func (m *cartMetrics) IncCartCleared(status string) {
	m.cartCleared.WithLabelValues(status).Inc()
}

func (m *cartMetrics) IncCartCheckout(status string) {
	m.cartCheckout.WithLabelValues(status).Inc()
}

func (m *cartMetrics) IncCartConsumed(status string) {
	m.cartConsumed.WithLabelValues(status).Inc()
}
