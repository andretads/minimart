// Package application is the payment domain's use-case layer. It depends on the
// payment repository, the provider, and the idempotency store through their
// ports - never on concrete infrastructure - so tests can pass fakes for all.
//
// CreatePayment is a synchronous use case: it creates the payment, calls the
// provider, persists the terminal state, and records each change as a domain
// event the repository writes to the transactional outbox.
package application

import (
	"context"
	"log/slog"
	"minimart/internal/payment/domain"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"time"

	orderclient "minimart/internal/order/presentation/module/client"
)

// metricsScope labels the payment domain's idempotency metrics.
const metricsScope = "payment"

// reservationTTL bounds how long a single request may hold the idempotency
// reservation while it talks to the provider. It must exceed normal processing
// time; the database's unique index still guards against duplicates if it does
// not.
const reservationTTL = time.Minute

// OrderReader is the narrow cross-domain port payment needs from the order
// domain: loading an order to verify its owner, total, and currency before a
// payment is charged. It is satisfied by contracts.Modules and by the published
// order client.
type OrderReader interface {
	GetOrder(ctx context.Context, id string) (orderclient.Order, error)
}

// Service is the payment application layer.
type Service struct {
	repo           domain.Repository
	orderReader    OrderReader
	idem           idempotency.Store
	provider       domain.Provider
	m              metrics.Metrics
	idempotencyTTL time.Duration
	idemCoord      *idempotency.Coordinator[domain.Payment]
}

// NewService wires the service over its ports. idempotencyTTL is how long a
// completed payment's idempotency key is remembered.
func NewService(repo domain.Repository, orderReader OrderReader, idem idempotency.Store, provider domain.Provider, m metrics.Metrics, idempotencyTTL time.Duration) *Service {
	s := &Service{repo: repo, orderReader: orderReader, idem: idem, provider: provider, m: m, idempotencyTTL: idempotencyTTL}
	s.idemCoord = &idempotency.Coordinator[domain.Payment]{
		Store:          idem,
		ReservationTTL: reservationTTL,
		CompletionTTL:  idempotencyTTL,
		LoadByID:       func(ctx context.Context, id string) (domain.Payment, error) { return repo.GetByID(ctx, id) },
		LoadByKey: func(ctx context.Context, key, fp string) (domain.Payment, bool, error) {
			return s.lookupByKey(ctx, key, fp)
		},
		NotFoundErr:   domain.ErrPaymentNotFound,
		InProgressErr: ErrPaymentInProgress,
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
