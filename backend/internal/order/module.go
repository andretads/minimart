package order

import (
	"context"
	"log/slog"
	"minimart/conf"
	"minimart/internal/order/application"
	"minimart/internal/order/infrastructure/autocancel"
	"minimart/internal/order/infrastructure/postgres"
	orderhttp "minimart/internal/order/presentation/http"
	ordermessage "minimart/internal/order/presentation/message"
	ordermodule "minimart/internal/order/presentation/module"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	outboxpkg "minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"minimart/pkg/worker"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Module wires the order domain. It holds the shared contracts registry (to call
// other domains), the DB pools, the Kafka publisher, and the idempotency store.
type Module struct {
	registry         *contracts.Contracts
	readPool         *pgxpool.Pool
	writePool        *pgxpool.Pool
	m                metrics.Metrics
	idem             idempotency.Store
	idempotencyTTL   time.Duration
	svc              *application.Service
	handlers         *orderhttp.Handlers
	autoCanceller    *application.AutoCanceller
	autoCancelRunner *autocancel.Runner
}

var _ module.Module = (*Module)(nil)

func NewModule(registry *contracts.Contracts, readPool, writePool *pgxpool.Pool, m metrics.Metrics, idem idempotency.Store, idempotencyTTL time.Duration) *Module {
	return &Module{
		registry:       registry,
		readPool:       readPool,
		writePool:      writePool,
		m:              m,
		idem:           idem,
		idempotencyTTL: idempotencyTTL,
	}
}

func (m *Module) Name() string { return "order" }

func (m *Module) Init(ctx context.Context) error {
	store := outboxpg.NewStore(m.writePool)
	repo := postgres.NewOrderRepository(m.readPool, m.writePool, outboxpkg.NewPublisher(store))
	m.svc = application.NewService(repo, m.registry, m.idem, m.m, m.idempotencyTTL)
	m.handlers = orderhttp.NewHandlers(m.svc)
	m.autoCanceller = application.NewAutoCanceller(repo, m.m, conf.OrderAutoCancelBatchSize(), slog.Default())
	m.autoCancelRunner = autocancel.New(m.autoCanceller, conf.OrderAutoCancelInterval(), slog.Default())
	return nil
}

// RegisterContracts publishes the OrderClient contract for cross-domain order placement.
func (m *Module) RegisterContracts(ctx context.Context, c *contracts.Contracts) error {
	c.OrderClient = ordermodule.New(m.svc)
	return nil
}

func (m *Module) RegisterHTTP(ctx context.Context, mux *http.ServeMux) error {
	m.handlers.Register(mux)
	return nil
}

// RegisterWorkers wires the order domain's Kafka consumers - its own confirmation
// plus the payment outcomes that update order status - and the auto-cancel runner.
func (m *Module) RegisterWorkers(ctx context.Context, w *worker.Worker) error {
	ordermessage.Workers(ctx, w, m.svc)
	if m.autoCancelRunner != nil {
		w.AttachBackground(ctx, m.autoCancelRunner.Run)
	}
	return nil
}
