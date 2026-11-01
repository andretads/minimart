package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// categoryMetrics carries the category-domain collectors.
type categoryMetrics struct {
	categoryCreated *prometheus.CounterVec
	categoryUpdated *prometheus.CounterVec
	categoryDeleted *prometheus.CounterVec
}

// newCategoryMetrics builds and registers the category collectors.
func newCategoryMetrics(reg prometheus.Registerer) categoryMetrics {
	m := categoryMetrics{
		// Dashboard: sum by (status) (rate(minimart_category_created_total[5m]))
		categoryCreated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "category",
			Name:      "created_total",
			Help:      "Total number of categories created by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_category_updated_total[5m]))
		categoryUpdated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "category",
			Name:      "updated_total",
			Help:      "Total number of categories updated by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_category_deleted_total[5m]))
		categoryDeleted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "category",
			Name:      "deleted_total",
			Help:      "Total number of categories deleted by result.",
		}, []string{"status"}),
	}
	reg.MustRegister(m.categoryCreated, m.categoryUpdated, m.categoryDeleted)
	return m
}

func (m *categoryMetrics) IncCategoryCreated(status string) {
	m.categoryCreated.WithLabelValues(status).Inc()
}

func (m *categoryMetrics) IncCategoryUpdated(status string) {
	m.categoryUpdated.WithLabelValues(status).Inc()
}

func (m *categoryMetrics) IncCategoryDeleted(status string) {
	m.categoryDeleted.WithLabelValues(status).Inc()
}
