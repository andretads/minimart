package shipping

import (
	"context"
	"minimart/internal/customer/presentation/module/client"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/infrastructure/postgres"
	shippingmodule "minimart/internal/shipping/presentation/module"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"minimart/pkg/outbox"
	"minimart/pkg/worker"
	"net/http"
	"time"

	shippinghttp "minimart/internal/shipping/presentation/http"
	shippingmessage "minimart/internal/shipping/presentation/message"
	outboxpg "minimart/pkg/outbox/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	registry       *contracts.Contracts
	readPool       *pgxpool.Pool
	writePool      *pgxpool.Pool
	m              metrics.Metrics
	idem           idempotency.Store
	idempotencyTTL time.Duration
	svc            *application.Service
	handlers       *shippinghttp.Handlers
	customer       client.CustomerClient
}

var _ module.Module = (*Module)(nil)

func NewModule(
	registry *contracts.Contracts,
	readPool, writePool *pgxpool.Pool,
	m metrics.Metrics,
	idem idempotency.Store,
	idempotencyTTL time.Duration,
) *Module {
	return &Module{
		registry:       registry,
		readPool:       readPool,
		writePool:      writePool,
		m:              m,
		idem:           idem,
		idempotencyTTL: idempotencyTTL,
	}
}

func (m *Module) Name() string { return "shipping" }

func (m *Module) Init(ctx context.Context) error {
	store := outboxpg.NewStore(m.writePool)
	repo := postgres.NewRepository(m.readPool, m.writePool, outbox.NewPublisher(store))
	m.svc = application.NewService(repo, m.registry, m.idem, m.m, m.idempotencyTTL)
	return nil
}

func (m *Module) RegisterContracts(ctx context.Context, c *contracts.Contracts) error {
	m.customer = c.CustomerClient
	m.handlers = shippinghttp.NewHandlers(m.svc, m.customer)
	c.ShippingClient = shippingmodule.New(m.svc)
	return nil
}

func (m *Module) RegisterHTTP(ctx context.Context, mux *http.ServeMux) error {
	m.handlers.Register(mux)
	return nil
}

func (m *Module) RegisterWorkers(ctx context.Context, w *worker.Worker) error {
	if w != nil {
		shippingmessage.Workers(ctx, w, m.svc)
	}
	return nil
}
