package wishlist

import (
	"context"
	"minimart/internal/wishlist/application"
	wishlistinfra "minimart/internal/wishlist/infrastructure/mongo"
	wishlisthttp "minimart/internal/wishlist/presentation/http"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"minimart/pkg/worker"
	"net/http"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
)

type Module struct {
	db       *mongodriver.Database
	reg      *contracts.Contracts
	m        metrics.Metrics
	svc      *application.Service
	handlers *wishlisthttp.Handlers
}

var _ module.Module = (*Module)(nil)

func NewModule(db *mongodriver.Database, reg *contracts.Contracts, m metrics.Metrics) *Module {
	return &Module{db: db, reg: reg, m: m}
}

func (m *Module) Name() string { return "wishlist" }

func (m *Module) Init(ctx context.Context) error {
	repo := wishlistinfra.NewRepository(m.db)
	if err := repo.EnsureIndexes(ctx); err != nil {
		return err
	}
	m.svc = application.NewService(repo, m.reg, m.m)
	m.handlers = wishlisthttp.NewHandlers(m.svc)
	return nil
}

func (m *Module) RegisterContracts(ctx context.Context, c *contracts.Contracts) error {
	return nil
}

func (m *Module) RegisterHTTP(ctx context.Context, mux *http.ServeMux) error {
	m.handlers.Register(mux)
	return nil
}

func (m *Module) RegisterWorkers(ctx context.Context, w *worker.Worker) error {
	return nil
}
