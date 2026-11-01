package application

import (
	"context"
	"log/slog"
	"minimart/internal/order/domain"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"minimart/pkg/module/contracts"
	"time"
)

// metricsScope labels the order domain's idempotency metrics.
const metricsScope = "order"

// reservationTTL bounds how long a single placement may hold the idempotency
// reservation while it reserves stock and saves the order. The database's unique
// index still guards against duplicates if it elapses.
const reservationTTL = time.Minute

// Service is the order application layer. It depends on the order repository port,
// on other domains through the contracts.Modules INTERFACE, and on the metrics
// recorder - never on concrete infrastructure. The repository persists the order's
// domain events to the transactional outbox in the same transaction as the order,
// so this layer no longer talks to Kafka directly. Depending on interfaces is what
// lets tests pass fakes for all of them.
type Service struct {
	repo           domain.Repository
	modules        contracts.Modules
	idem           idempotency.Store
	m              metrics.Metrics
	idempotencyTTL time.Duration
	idemCoord      *idempotency.Coordinator[domain.Order]
}

// NewService wires the service over its ports. idempotencyTTL is how long a
// completed order's idempotency key is remembered.
func NewService(repo domain.Repository, modules contracts.Modules, idem idempotency.Store, m metrics.Metrics, idempotencyTTL time.Duration) *Service {
	s := &Service{repo: repo, modules: modules, idem: idem, m: m, idempotencyTTL: idempotencyTTL}
	s.idemCoord = &idempotency.Coordinator[domain.Order]{
		Store:          idem,
		ReservationTTL: reservationTTL,
		CompletionTTL:  idempotencyTTL,
		LoadByID:       func(ctx context.Context, id string) (domain.Order, error) { return repo.GetByID(ctx, id) },
		LoadByKey: func(ctx context.Context, key, fp string) (domain.Order, bool, error) {
			return s.lookupByKey(ctx, key, fp)
		},
		NotFoundErr:   domain.ErrOrderNotFound,
		InProgressErr: ErrOrderInProgress,
		ConflictErr:   domain.ErrIdempotencyConflict,
		OnMiss:        func(ctx context.Context) { s.m.IncIdempotency(metricsScope, "miss") },
		OnConflict:    func(ctx context.Context) { s.m.IncIdempotency(metricsScope, "conflict") },
		OnInProgress:  func(ctx context.Context) { s.m.IncIdempotency(metricsScope, "in_progress") },
		OnReplay:      func(ctx context.Context) { s.m.IncIdempotency(metricsScope, "replay") },
		OnStoreError: func(ctx context.Context, err error) {
			slog.WarnContext(ctx, "idempotency store unavailable; falling back to database", "scope", metricsScope, "error", err)
			s.m.IncIdempotency(metricsScope, "error")
		},
	}
	return s
}
