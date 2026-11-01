// Package customer wires the customer domain into the modular monolith. It is
// the domain's composition root: the one place that names concrete adapters.
package customer

import (
	"context"
	"minimart/internal/customer/application"
	"minimart/internal/customer/infrastructure/postgres"
	customerhttp "minimart/internal/customer/presentation/http"
	customermodule "minimart/internal/customer/presentation/module"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"minimart/pkg/worker"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Module is the customer domain's self-wiring unit. It plays the standard roles:
// a server role (exposes HTTP endpoints for customer profile management) and a
// contract publisher (CustomerClient for other domains to look up customers).
// It does not run background workers.
type Module struct {
	// readPool is the replica pool for read queries.
	readPool *pgxpool.Pool
	// writePool is the primary pool for write queries.
	writePool *pgxpool.Pool
	// m is the metrics sink for recording domain events.
	m metrics.Metrics
	// svc is the application service with business logic.
	svc *application.Service
	// handlers are the HTTP handlers for the customer REST API.
	handlers *customerhttp.Handlers
}

var _ module.Module = (*Module)(nil)

func NewModule(readPool, writePool *pgxpool.Pool, m metrics.Metrics) *Module {
	return &Module{readPool: readPool, writePool: writePool, m: m}
}

func (m *Module) Name() string { return "customer" }

// Init chooses the concrete storage adapter (Postgres). Swapping storage is
// these few lines; no other layer is touched. (Earlier stages used memory.New() here.)
func (m *Module) Init(ctx context.Context) error {
	repo := postgres.NewCustomerRepository(m.readPool, m.writePool)
	m.svc = application.NewService(repo, m.m)
	m.handlers = customerhttp.NewHandlers(m.svc)
	return nil
}

// RegisterContracts publishes the CustomerClient contract so other domains
// (e.g., order) can look up or create customers without importing customer's
// application or infrastructure layers.
func (m *Module) RegisterContracts(ctx context.Context, c *contracts.Contracts) error {
	c.CustomerClient = customermodule.New(m.svc)
	return nil
}

func (m *Module) RegisterHTTP(ctx context.Context, mux *http.ServeMux) error {
	m.handlers.Register(mux)
	return nil
}

// RegisterWorkers is a no-op: the customer domain has no background consumers.
func (m *Module) RegisterWorkers(ctx context.Context, w *worker.Worker) error { return nil }
