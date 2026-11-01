package address

import (
	"context"
	"minimart/internal/address/application"
	"minimart/internal/address/infrastructure/postgres"
	addresshttp "minimart/internal/address/presentation/http"
	addressmodule "minimart/internal/address/presentation/module"
	"minimart/internal/customer/presentation/module/client"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"minimart/pkg/worker"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	readPool  *pgxpool.Pool
	writePool *pgxpool.Pool
	m         metrics.Metrics
	svc       *application.Service
	handlers  *addresshttp.Handlers
	customer  client.CustomerClient
}

var _ module.Module = (*Module)(nil)

func NewModule(readPool, writePool *pgxpool.Pool, m metrics.Metrics) *Module {
	return &Module{readPool: readPool, writePool: writePool, m: m}
}

func (m *Module) Name() string { return "address" }

func (m *Module) Init(ctx context.Context) error {
	repo := postgres.NewAddressRepository(m.readPool, m.writePool)
	m.svc = application.NewService(repo, m.m)
	return nil
}

func (m *Module) RegisterContracts(ctx context.Context, c *contracts.Contracts) error {
	m.customer = c.CustomerClient
	m.handlers = addresshttp.NewHandlers(m.svc, m.customer)
	c.AddressClient = addressmodule.New(m.svc)
	return nil
}

func (m *Module) RegisterHTTP(ctx context.Context, mux *http.ServeMux) error {
	m.handlers.Register(mux)
	return nil
}

func (m *Module) RegisterWorkers(ctx context.Context, w *worker.Worker) error {
	return nil
}
