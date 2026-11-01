package application

import (
	"minimart/internal/wishlist/domain"
	"minimart/pkg/metrics"
	"minimart/pkg/module/contracts"
)

// Service is the wishlist application layer. It depends on the wishlist repository port,
// on other domains through the contracts.Modules interface, and on the metrics
// recorder - never on concrete infrastructure.
type Service struct {
	repo    domain.Repository
	modules contracts.Modules
	m       metrics.Metrics
}

// NewService wires the service over its ports.
func NewService(repo domain.Repository, modules contracts.Modules, m metrics.Metrics) *Service {
	return &Service{repo: repo, modules: modules, m: m}
}
