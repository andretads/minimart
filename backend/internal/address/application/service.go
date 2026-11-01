package application

import (
	"minimart/internal/address/domain"
	"minimart/pkg/metrics"
)

// Service is the address application layer. It depends on the persistence
// port - never on concrete infrastructure - so tests can pass fakes.
type Service struct {
	repo domain.Repository
	m    metrics.Metrics
}

// NewService wires the service over its persistence port.
func NewService(repo domain.Repository, m metrics.Metrics) *Service {
	return &Service{repo: repo, m: m}
}
