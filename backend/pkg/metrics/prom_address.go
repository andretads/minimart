package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// addressMetrics carries the address-domain collectors.
type addressMetrics struct {
	addressCreated    *prometheus.CounterVec
	addressFetched    *prometheus.CounterVec
	addressListed     *prometheus.CounterVec
	addressUpdated    *prometheus.CounterVec
	addressDeleted    *prometheus.CounterVec
	addressDefaultSet *prometheus.CounterVec
}

// newAddressMetrics builds and registers the address collectors.
func newAddressMetrics(reg prometheus.Registerer) addressMetrics {
	m := addressMetrics{
		// Dashboard: sum by (status) (rate(minimart_address_created_total[5m]))
		addressCreated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "address",
			Name:      "created_total",
			Help:      "Total number of addresses created by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_address_fetched_total[5m]))
		addressFetched: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "address",
			Name:      "fetched_total",
			Help:      "Total number of addresses fetched by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_address_listed_total[5m]))
		addressListed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "address",
			Name:      "listed_total",
			Help:      "Total number of address listings by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_address_updated_total[5m]))
		addressUpdated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "address",
			Name:      "updated_total",
			Help:      "Total number of addresses updated by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_address_deleted_total[5m]))
		addressDeleted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "address",
			Name:      "deleted_total",
			Help:      "Total number of addresses deleted by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_address_default_set_total[5m]))
		addressDefaultSet: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "address",
			Name:      "default_set_total",
			Help:      "Total number of address default set operations by result.",
		}, []string{"status"}),
	}
	reg.MustRegister(
		m.addressCreated,
		m.addressFetched,
		m.addressListed,
		m.addressUpdated,
		m.addressDeleted,
		m.addressDefaultSet,
	)
	return m
}

func (m *addressMetrics) IncAddressCreated(status string) {
	m.addressCreated.WithLabelValues(status).Inc()
}

func (m *addressMetrics) IncAddressFetched(status string) {
	m.addressFetched.WithLabelValues(status).Inc()
}

func (m *addressMetrics) IncAddressListed(status string) {
	m.addressListed.WithLabelValues(status).Inc()
}

func (m *addressMetrics) IncAddressUpdated(status string) {
	m.addressUpdated.WithLabelValues(status).Inc()
}

func (m *addressMetrics) IncAddressDeleted(status string) {
	m.addressDeleted.WithLabelValues(status).Inc()
}

func (m *addressMetrics) IncAddressDefaultSet(status string) {
	m.addressDefaultSet.WithLabelValues(status).Inc()
}
