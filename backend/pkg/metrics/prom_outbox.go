package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var outboxStates = []string{"pending", "locked", "delivered", "abandoned"}

// outboxMetrics carries the transactional-outbox backlog gauges. Topic/key
// labels are deliberately absent: the table is shared by all relays and record
// IDs are unbounded.
type outboxMetrics struct {
	outboxRecords   *prometheus.GaugeVec
	outboxOldestAge *prometheus.GaugeVec
}

// newOutboxMetrics builds and registers the outbox collectors.
func newOutboxMetrics(reg prometheus.Registerer) outboxMetrics {
	m := outboxMetrics{
		// Dashboard: sum by (state) (minimart_outbox_records)
		// Alert:     sum(minimart_outbox_records{state="abandoned"}) > 0
		outboxRecords: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "outbox",
			Name:      "records",
			Help:      "Current number of outbox records by state.",
		}, []string{"state"}),

		// Dashboard: max by (state) (minimart_outbox_oldest_record_age_seconds)
		// Alert:     max(minimart_outbox_oldest_record_age_seconds{state="pending"}) > 900
		outboxOldestAge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "outbox",
			Name:      "oldest_record_age_seconds",
			Help:      "Age in seconds of the oldest outbox record in each state.",
		}, []string{"state"}),
	}

	for _, state := range outboxStates {
		m.outboxRecords.WithLabelValues(state).Set(0)
		m.outboxOldestAge.WithLabelValues(state).Set(0)
	}

	reg.MustRegister(m.outboxRecords, m.outboxOldestAge)
	return m
}

func (m *outboxMetrics) SetOutboxRecords(state string, count int64) {
	m.outboxRecords.WithLabelValues(state).Set(float64(count))
}

func (m *outboxMetrics) SetOutboxOldestRecordAge(state string, age time.Duration) {
	m.outboxOldestAge.WithLabelValues(state).Set(age.Seconds())
}
