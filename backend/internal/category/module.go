// Package category wires the category domain into the modular monolith. It is
// the domain's composition root: the one place that names concrete adapters.
package category

import (
	"context"
	"minimart/internal/category/application"
	"minimart/internal/category/infrastructure/postgres"
	categoryhttp "minimart/internal/category/presentation/http"
	categorymodule "minimart/internal/category/presentation/module"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"minimart/pkg/worker"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Module is the category domain's self-wiring unit. It plays the server role
// (exposes HTTP endpoints for category management) and publishes the Category
// contract for cross-domain category lookups. It does not run background
// workers.
type Module struct {
	// readPool is the replica pool for read queries.
	readPool *pgxpool.Pool
	// writePool is the primary pool for write queries.
	writePool *pgxpool.Pool
	// m is the metrics sink for recording domain events.
	m metrics.Metrics
	// svc is the application service with business logic.
	svc *application.Service
	// handlers are the HTTP handlers for the category REST API.
	handlers *categoryhttp.Handlers
}

var _ module.Module = (*Module)(nil)

func NewModule(readPool, writePool *pgxpool.Pool, m metrics.Metrics) *Module {
	return &Module{readPool: readPool, writePool: writePool, m: m}
}

func (m *Module) Name() string { return "category" }

// Init chooses the concrete storage adapter (Postgres). Swapping storage is
// these few lines; no other layer is touched.
func (m *Module) Init(ctx context.Context) error {
	repo := postgres.NewCategoryRepository(m.readPool, m.writePool)
	m.svc = application.NewService(repo, m.m)
	m.handlers = categoryhttp.NewHandlers(m.svc)
	return nil
}

// RegisterContracts publishes the Category contract for cross-domain lookups.
func (m *Module) RegisterContracts(ctx context.Context, c *contracts.Contracts) error {
	c.CategoryClient = categorymodule.New(m.svc)
	return nil
}

func (m *Module) RegisterHTTP(ctx context.Context, mux *http.ServeMux) error {
	m.handlers.Register(mux)
	return nil
}

// RegisterWorkers is a no-op: the category domain has no background consumers.
func (m *Module) RegisterWorkers(ctx context.Context, w *worker.Worker) error { return nil }
