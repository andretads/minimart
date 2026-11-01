package application

import (
	"context"
	"minimart/internal/shipping/domain"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"minimart/pkg/module/contracts"
	"time"
)

// metricsScope labels the shipping domain's idempotency metrics.
const metricsScope = "shipping"

// reservationTTL bounds how long a single order-paid handling may hold the
// idempotency reservation while it creates the shipment. The database's unique
// index on shipments.order_id still guards against duplicates if it elapses.
const reservationTTL = time.Minute

// Service is the shipping application layer. It depends on the shipping
// repository port, on other domains through the contracts.Modules INTERFACE,
// on the idempotency store, and on the metrics recorder - never on concrete
// infrastructure. The repository persists the shipment's domain events to the
// transactional outbox in the same transaction as the shipment, so this layer
// never talks to Kafka directly. Depending on interfaces is what lets tests
// pass fakes for all of them.
type Service struct {
	repo           domain.Repository
	modules        contracts.Modules
	idem           idempotency.Store
	m              metrics.Metrics
	idempotencyTTL time.Duration
	quoteCoord     *idempotency.Coordinator[domain.ShippingQuote]
	shipmentCoord  *idempotency.Coordinator[domain.Shipment]
}

// NewService wires the service over its ports. idempotencyTTL is how long a
// completed shipment's idempotency key is remembered.
func NewService(repo domain.Repository, modules contracts.Modules, idem idempotency.Store, m metrics.Metrics, idempotencyTTL time.Duration) *Service {
	s := &Service{repo: repo, modules: modules, idem: idem, m: m, idempotencyTTL: idempotencyTTL}
	s.quoteCoord = &idempotency.Coordinator[domain.ShippingQuote]{
		Store:          idem,
		ReservationTTL: reservationTTL,
		CompletionTTL:  idempotencyTTL,
		LoadByID:       func(ctx context.Context, id string) (domain.ShippingQuote, error) { return repo.GetQuote(ctx, id) },
		LoadByKey: func(ctx context.Context, key, fp string) (domain.ShippingQuote, bool, error) {
			return s.lookupQuoteByKey(ctx, key, fp)
		},
		NotFoundErr:   domain.ErrQuoteNotFound,
		InProgressErr: domain.ErrQuoteInProgress,
		ConflictErr:   domain.ErrIdempotencyConflict,
		OnMiss:        func(ctx context.Context) { s.m.IncIdempotency(metricsScope, "miss") },
		OnConflict:    func(ctx context.Context) { s.m.IncIdempotency(metricsScope, "conflict") },
		OnInProgress:  func(ctx context.Context) { s.m.IncIdempotency(metricsScope, "in_progress") },
		OnReplay:      func(ctx context.Context) { s.m.IncIdempotency(metricsScope, "replay") },
		OnStoreError:  func(ctx context.Context, err error) { s.m.IncIdempotency(metricsScope, "error") },
	}
	s.shipmentCoord = &idempotency.Coordinator[domain.Shipment]{
		Store:          idem,
		ReservationTTL: reservationTTL,
		CompletionTTL:  idempotencyTTL,
		LoadByID:       func(ctx context.Context, id string) (domain.Shipment, error) { return repo.GetShipment(ctx, id) },
		LoadByKey: func(ctx context.Context, key, fp string) (domain.Shipment, bool, error) {
			orderID := key
			const prefix = "shipping:shipment:"
			if len(key) > len(prefix) && key[:len(prefix)] == prefix {
				orderID = key[len(prefix):]
			}
			return s.lookupShipment(ctx, orderID, fp)
		},
		NotFoundErr:   domain.ErrShipmentNotFound,
		InProgressErr: domain.ErrShipmentInProgress,
		ConflictErr:   domain.ErrIdempotencyConflict,
		OnMiss:        func(ctx context.Context) { s.m.IncIdempotency(metricsScope, "miss") },
		OnConflict:    func(ctx context.Context) { s.m.IncIdempotency(metricsScope, "conflict") },
		OnInProgress:  func(ctx context.Context) { s.m.IncIdempotency(metricsScope, "in_progress") },
		OnReplay:      func(ctx context.Context) { s.m.IncIdempotency(metricsScope, "replay") },
		OnStoreError:  func(ctx context.Context, err error) { s.m.IncIdempotency(metricsScope, "error") },
	}
	return s
}
