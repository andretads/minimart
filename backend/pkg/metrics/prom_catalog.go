package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// catalogMetrics carries the catalog-domain collectors: products, stock, and
// review moderation.
type catalogMetrics struct {
	productCreated *prometheus.CounterVec
	productUpdated *prometheus.CounterVec
	stockReserved  *prometheus.CounterVec
	stockReleased  *prometheus.CounterVec
}

// newCatalogMetrics builds and registers the catalog collectors.
func newCatalogMetrics(reg prometheus.Registerer) catalogMetrics {
	m := catalogMetrics{
		// Dashboard: sum by (status) (rate(minimart_product_created_total[5m]))
		productCreated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "product",
			Name:      "created_total",
			Help:      "Total number of products created by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_product_updated_total[5m]))
		productUpdated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "product",
			Name:      "updated_total",
			Help:      "Total number of products updated by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_stock_reserved_total[5m]))
		stockReserved: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "stock",
			Name:      "reserved_total",
			Help:      "Total number of stock reservations by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_stock_released_total[5m]))
		stockReleased: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "stock",
			Name:      "released_total",
			Help:      "Total number of stock releases by result.",
		}, []string{"status"}),
	}
	reg.MustRegister(m.productCreated, m.productUpdated, m.stockReserved, m.stockReleased)
	return m
}

func (m *catalogMetrics) IncProductCreated(status string) {
	m.productCreated.WithLabelValues(status).Inc()
}

func (m *catalogMetrics) IncProductUpdated(status string) {
	m.productUpdated.WithLabelValues(status).Inc()
}

// IncReviewAdded reuses productUpdated for now; reviews need their own series
// once dashboards distinguish them.
func (m *catalogMetrics) IncReviewAdded(status string) {
	m.productUpdated.WithLabelValues(status).Inc()
}

// IncReviewModerated reuses productUpdated for now; see IncReviewAdded.
func (m *catalogMetrics) IncReviewModerated(decision string) {
	m.productUpdated.WithLabelValues(decision).Inc()
}

func (m *catalogMetrics) IncStockReserved(status string) {
	m.stockReserved.WithLabelValues(status).Inc()
}

func (m *catalogMetrics) IncStockReleased(status string) {
	m.stockReleased.WithLabelValues(status).Inc()
}
