package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// avatarMetrics carries the avatar-domain collectors.
type avatarMetrics struct {
	avatarUploaded *prometheus.CounterVec
	avatarFetched  *prometheus.CounterVec
}

// newAvatarMetrics builds and registers the avatar collectors.
func newAvatarMetrics(reg prometheus.Registerer) avatarMetrics {
	m := avatarMetrics{
		// Dashboard: sum by (status) (rate(minimart_avatar_uploaded_total[5m]))
		avatarUploaded: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "avatar",
			Name:      "uploaded_total",
			Help:      "Total number of avatar uploads by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_avatar_fetched_total[5m]))
		avatarFetched: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "avatar",
			Name:      "fetched_total",
			Help:      "Total number of avatar fetches by result.",
		}, []string{"status"}),
	}
	reg.MustRegister(m.avatarUploaded, m.avatarFetched)
	return m
}

func (m *avatarMetrics) IncAvatarUploaded(status string) {
	m.avatarUploaded.WithLabelValues(status).Inc()
}

func (m *avatarMetrics) IncAvatarFetched(status string) {
	m.avatarFetched.WithLabelValues(status).Inc()
}
