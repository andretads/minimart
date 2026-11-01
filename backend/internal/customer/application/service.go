package application

import (
	"minimart/internal/customer/domain"
	"minimart/pkg/metrics"
)

// metricsScope labels the customer domain's metrics.
const metricsScope = "customer" //nolint:unused // defined for consistency; used in use case files when idempotency pattern is adopted

// Service is the customer application layer. It depends on the persistence
// port - never on concrete infrastructure - so tests can pass fakes.
type Service struct {
	repo domain.Repository
	m    metrics.Metrics
}

// NewService wires the service over its persistence port.
func NewService(repo domain.Repository, m metrics.Metrics) *Service {
	return &Service{repo: repo, m: m}
}
