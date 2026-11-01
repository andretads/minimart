package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// notificationMetrics carries the notification-domain collectors.
type notificationMetrics struct {
	notificationSent  *prometheus.CounterVec
	notificationPrefs *prometheus.CounterVec
	notificationMuted *prometheus.CounterVec
}

// newNotificationMetrics builds and registers the notification collectors.
func newNotificationMetrics(reg prometheus.Registerer) notificationMetrics {
	m := notificationMetrics{
		// Dashboard: sum by (status) (rate(minimart_notification_sent_total[5m]))
		notificationSent: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "notification",
			Name:      "sent_total",
			Help:      "Total number of order confirmations sent by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_notification_preferences_updated_total[5m]))
		notificationPrefs: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "notification",
			Name:      "preferences_updated_total",
			Help:      "Total number of notification preference updates by result.",
		}, []string{"status"}),

		// Dashboard: sum by (type) (rate(minimart_notification_suppressed_total[5m]))
		notificationMuted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "notification",
			Name:      "suppressed_total",
			Help:      "Total number of notifications suppressed by user preferences.",
		}, []string{"type"}),
	}
	reg.MustRegister(m.notificationSent, m.notificationPrefs, m.notificationMuted)
	return m
}

func (m *notificationMetrics) IncNotificationSent(status string) {
	m.notificationSent.WithLabelValues(status).Inc()
}

func (m *notificationMetrics) IncNotificationPreferencesUpdated(status string) {
	m.notificationPrefs.WithLabelValues(status).Inc()
}

func (m *notificationMetrics) IncNotificationSuppressed(notificationType string) {
	m.notificationMuted.WithLabelValues(notificationType).Inc()
}
