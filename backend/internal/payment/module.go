// Package payment wires the payment domain: its application service, Postgres
// repository, HTTP handlers, the order-cancelled consumer that refunds
// captured payments, and the outbox relay that publishes its events.
package payment

import (
	"context"
	"log/slog"
	"minimart/conf"
	"minimart/internal/payment/application"
	"minimart/internal/payment/domain"
	"minimart/internal/payment/infrastructure/postgres"
	"minimart/internal/payment/infrastructure/reconciliation"
	"minimart/pkg/idempotency"
	"minimart/pkg/kafka"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"minimart/pkg/worker"
	"net/http"
	"time"

	paymentoutbox "minimart/internal/payment/infrastructure/outbox"
	paymenthttp "minimart/internal/payment/presentation/http"
	paymentmessage "minimart/internal/payment/presentation/message"
	outboxpkg "minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Module wires the payment domain.
type Module struct {
	registry             *contracts.Contracts
	readPool             *pgxpool.Pool
	writePool            *pgxpool.Pool
	publisher            kafka.Publisher
	m                    metrics.Metrics
	idem                 idempotency.Store
	provider             domain.Provider
	idempotencyTTL       time.Duration
	outboxSettings       paymentoutbox.Settings
	svc                  *application.Service
	handlers             *paymenthttp.Handlers
	relay                *paymentoutbox.Relay
	reconciler           *application.Reconciler
	reconciliationRunner *reconciliation.Runner
}

var _ module.Module = (*Module)(nil)

// NewModule builds the payment module. idem and provider are injected so the
// composition root chooses their adapters (Redis store, fake provider).
func NewModule(registry *contracts.Contracts, readPool, writePool *pgxpool.Pool, publisher kafka.Publisher, m metrics.Metrics, idem idempotency.Store, provider domain.Provider, idempotencyTTL time.Duration, outboxSettings paymentoutbox.Settings) *Module {
	return &Module{
		registry:       registry,
		readPool:       readPool,
		writePool:      writePool,
		publisher:      publisher,
		m:              m,
		idem:           idem,
		provider:       provider,
		idempotencyTTL: idempotencyTTL,
		outboxSettings: outboxSettings,
	}
}

// Name returns the module's name.
func (m *Module) Name() string { return "payment" }

// Init builds the module's repository, service, handlers, and relay.
func (m *Module) Init(ctx context.Context) error {
	store := outboxpg.NewStore(m.writePool)
	repo := postgres.NewPaymentRepository(m.readPool, m.writePool, outboxpkg.NewPublisher(store))
	m.svc = application.NewService(repo, m.registry, m.idem, m.provider, m.m, m.idempotencyTTL)
	m.handlers = paymenthttp.NewHandlers(m.svc)
	m.relay = paymentoutbox.NewRelay(store, m.publisher, paymentoutbox.MachineID(), m.outboxSettings)
	m.reconciler = application.NewReconciler(m.svc, conf.PaymentReconcileAge(), conf.PaymentReconcileBatchSize(), slog.Default())
	m.reconciliationRunner = reconciliation.New(m.reconciler, conf.PaymentReconcileInterval(), slog.Default())
	return nil
}

// RegisterContracts is a no-op: the payment domain publishes no contract of its own.
func (m *Module) RegisterContracts(ctx context.Context, c *contracts.Contracts) error { return nil }

// RegisterHTTP wires the payment routes onto the mux.
func (m *Module) RegisterHTTP(ctx context.Context, mux *http.ServeMux) error {
	m.handlers.Register(mux)
	return nil
}

// RegisterWorkers starts the outbox relay that publishes this domain's events
// and consumes the order-cancelled topic to refund captured payments.
func (m *Module) RegisterWorkers(ctx context.Context, w *worker.Worker) error {
	if w != nil {
		paymentmessage.Workers(ctx, w, m.svc)
		w.AttachBackground(ctx, m.relay.Run)
		if m.reconciliationRunner != nil {
			w.AttachBackground(ctx, m.reconciliationRunner.Run)
		}
	}
	return nil
}
