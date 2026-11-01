package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// customerMetrics carries the customer-domain collectors.
type customerMetrics struct {
	customerEnsured     *prometheus.CounterVec
	customerCreated     *prometheus.CounterVec
	customerUpdated     *prometheus.CounterVec
	customerDeactivated *prometheus.CounterVec
}

// newCustomerMetrics builds and registers the customer collectors.
func newCustomerMetrics(reg prometheus.Registerer) customerMetrics {
	m := customerMetrics{
		// Dashboard: sum by (status) (rate(minimart_customer_created_total[5m]))
		customerCreated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "customer",
			Name:      "created_total",
			Help:      "Total number of customers created by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_customer_updated_total[5m]))
		customerUpdated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "customer",
			Name:      "updated_total",
			Help:      "Total number of customers updated by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_customer_deactivated_total[5m]))
		customerDeactivated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "customer",
			Name:      "deactivated_total",
			Help:      "Total number of customers deactivated by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_customer_ensured_total[5m]))
		customerEnsured: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "customer",
			Name:      "ensured_total",
			Help:      "Total number of customer ensure operations by result.",
		}, []string{"status"}),
	}
	reg.MustRegister(m.customerEnsured, m.customerCreated, m.customerUpdated, m.customerDeactivated)
	return m
}

func (m *customerMetrics) IncCustomerEnsured(status string) {
	m.customerEnsured.WithLabelValues(status).Inc()
}

func (m *customerMetrics) IncCustomerCreated(status string) {
	m.customerCreated.WithLabelValues(status).Inc()
}

func (m *customerMetrics) IncCustomerUpdated(status string) {
	m.customerUpdated.WithLabelValues(status).Inc()
}

func (m *customerMetrics) IncCustomerDeactivated(status string) {
	m.customerDeactivated.WithLabelValues(status).Inc()
}
