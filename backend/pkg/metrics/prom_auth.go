package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// authMetrics carries the auth- and authz-domain collectors.
type authMetrics struct {
	accountCreated    *prometheus.CounterVec
	authenticated     *prometheus.CounterVec
	passwordChanged   *prometheus.CounterVec
	tokenRefreshed    *prometheus.CounterVec
	roleAssigned      *prometheus.CounterVec
	permissionsLoaded *prometheus.CounterVec
	authzDenied       *prometheus.CounterVec
}

// newAuthMetrics builds and registers the auth collectors.
func newAuthMetrics(reg prometheus.Registerer) authMetrics {
	m := authMetrics{
		// Dashboard: sum by (status) (rate(minimart_account_created_total[5m]))
		accountCreated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "auth",
			Name:      "account_created_total",
			Help:      "Total number of accounts created by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_authenticated_total[5m]))
		authenticated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "auth",
			Name:      "authenticated_total",
			Help:      "Total number of authentications by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_auth_password_changed_total[5m]))
		passwordChanged: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "auth",
			Name:      "password_changed_total",
			Help:      "Total number of password changes by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_token_refreshed_total[5m]))
		tokenRefreshed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "auth",
			Name:      "token_refreshed_total",
			Help:      "Total number of token refreshes by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_role_assigned_total[5m]))
		roleAssigned: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "auth",
			Name:      "role_assigned_total",
			Help:      "Total number of role assignments by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_permissions_loaded_total[5m]))
		permissionsLoaded: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "auth",
			Name:      "permissions_loaded_total",
			Help:      "Total number of permission lookups by result.",
		}, []string{"status"}),

		// Dashboard: sum by (status) (rate(minimart_authz_denied_total[5m]))
		// Alert:     rate(minimart_authz_denied_total{status="forbidden"}[5m]) > 10
		authzDenied: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "authz",
			Name:      "denied_total",
			Help:      "Total number of authorization denials by result (forbidden, error).",
		}, []string{"status"}),
	}
	reg.MustRegister(
		m.accountCreated,
		m.authenticated,
		m.passwordChanged,
		m.tokenRefreshed,
		m.roleAssigned,
		m.permissionsLoaded,
		m.authzDenied,
	)
	return m
}

func (m *authMetrics) IncAccountCreated(status string) {
	m.accountCreated.WithLabelValues(status).Inc()
}

func (m *authMetrics) IncAuthenticated(status string) {
	m.authenticated.WithLabelValues(status).Inc()
}

func (m *authMetrics) IncPasswordChanged(status string) {
	m.passwordChanged.WithLabelValues(status).Inc()
}

func (m *authMetrics) IncTokenRefreshed(status string) {
	m.tokenRefreshed.WithLabelValues(status).Inc()
}

func (m *authMetrics) IncRoleAssigned(status string) {
	m.roleAssigned.WithLabelValues(status).Inc()
}

func (m *authMetrics) IncPermissionsLoaded(status string) {
	m.permissionsLoaded.WithLabelValues(status).Inc()
}

func (m *authMetrics) IncAuthzDenied(status string) {
	m.authzDenied.WithLabelValues(status).Inc()
}
