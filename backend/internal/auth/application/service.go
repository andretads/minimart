package application

import (
	"context"
	"minimart/internal/auth/domain"
	"minimart/internal/customer/presentation/module/client"
	"minimart/pkg/metrics"
)

const metricsScope = "auth"

type CustomerProvisioner interface {
	EnsureCustomer(ctx context.Context, userID string) (client.Customer, error)
}

type RoleAssigner interface {
	AssignDefaultRole(ctx context.Context, accountID string) error
}

type PermissionStore interface {
	PermissionsForAccount(ctx context.Context, accountID string) ([]string, error)
}

type Service struct {
	repo      domain.Repository
	customers CustomerProvisioner
	m         metrics.Metrics
}

func NewService(repo domain.Repository, customers CustomerProvisioner, m metrics.Metrics) *Service {
	return &Service{repo: repo, customers: customers, m: m}
}
