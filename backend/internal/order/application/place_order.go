package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	catalogclient "minimart/internal/catalog/presentation/module/client"
	"minimart/internal/order/domain"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"strconv"
)

// PlaceOrderRequest is the input to PlaceOrder. It carries NO prices or names:
// those are authoritative from the catalog, fetched below. The client only says
// which product and how many. IdempotencyKey makes a retry safe - a repeated
// request with the same key returns the original order.
type PlaceOrderRequest struct {
	UserID              string
	IdempotencyKey      string
	Items               []PlaceOrderItem
	ShippingQuoteID     string
	ShippingMethodID    string
	ShippingMethodName  string
	DestinationSnapshot domain.DestinationSnapshot
	ShippingCostCents   int64
}

type PlaceOrderItem struct {
	ProductID string
	Quantity  int
}

// PlaceOrder resolves each product's real price from the catalog (through the
// published contract), reserves its stock, builds and saves the order. The
// repository writes the order and its OrderPlaced event to the transactional
// outbox in the same transaction, so the event is published at least once by the
// outbox relay.
//
// It is idempotent: a repeated call with the same key and parameters replays the
// stored order; the same key with different parameters is a conflict. Redis is a
// fast cache in front of the database; if it is unavailable the flow falls back to
// the orders table's unique index, which is the authoritative guard.
func (s *Service) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (domain.Order, error) {
	order, err := s.placeOrder(ctx, req)
	if err != nil {
		s.m.IncOrderPlaced(metrics.StatusError)
		return order, err
	}
	s.m.IncOrderPlaced(metrics.StatusSuccess)
	s.m.ObserveOrderAmount(float64(order.TotalCents))
	return order, nil
}

func (s *Service) placeOrder(ctx context.Context, req PlaceOrderRequest) (domain.Order, error) {
	if req.IdempotencyKey == "" {
		return domain.Order{}, ErrIdempotencyKeyRequired
	}
	key := req.IdempotencyKey
	fp := requestFingerprint(req)

	// Fast path: has this key already produced an order?
	o, done, err := s.existing(ctx, key, fp)
	if err != nil {
		return domain.Order{}, err
	}
	if done {
		return o, nil
	}

	// Claim the key so a concurrent duplicate fails fast instead of placing a
	// second order. On a store error this fails open (returns true).
	if !s.reserve(ctx, key, fp) {
		o, done, err = s.existing(ctx, key, fp)
		if err != nil {
			return domain.Order{}, err
		}
		if done {
			return o, nil
		}
		return domain.Order{}, ErrOrderInProgress
	}

	order, err := s.process(ctx, req)
	if err != nil {
		// A different request won the race and created this key's order. Look it
		// up in the database, not the cache.
		if errors.Is(err, domain.ErrIdempotencyConflict) {
			o, done, lookupErr := s.lookupByKey(ctx, key, fp)
			if lookupErr != nil {
				return domain.Order{}, lookupErr
			}
			if done {
				return o, nil
			}
		}
		// Free the reservation so the caller may retry.
		s.release(ctx, key, fp)
		return domain.Order{}, err
	}

	s.complete(ctx, key, fp, order.ID)
	return order, nil
}

// process resolves products, reserves stock, and saves a single order. It releases
// the stock reservation if saving the order fails, so a failed placement does not
// leak held stock.
func (s *Service) process(ctx context.Context, req PlaceOrderRequest) (domain.Order, error) {
	if len(req.Items) == 0 {
		return domain.Order{}, domain.ErrNoItems
	}
	items := make([]domain.Item, 0, len(req.Items))
	lines := make([]catalogclient.LineItem, 0, len(req.Items))
	for _, it := range req.Items {
		if it.Quantity <= 0 {
			return domain.Order{}, domain.ErrInvalidQuantity
		}
		p, err := s.modules.GetProduct(ctx, it.ProductID)
		if err != nil {
			return domain.Order{}, fmt.Errorf("resolve product %s: %w", it.ProductID, err)
		}
		items = append(items, domain.Item{
			ProductID:      p.ID,
			Name:           p.Name,
			UnitPriceCents: p.PriceCents,
			Quantity:       it.Quantity,
		})
		lines = append(lines, catalogclient.LineItem{ProductID: p.ID, Quantity: it.Quantity})
	}
	cust, err := s.modules.EnsureCustomer(ctx, req.UserID)
	if err != nil {
		return domain.Order{}, err
	}
	order, err := domain.NewOrder(newID(), req.UserID, cust.ID, req.IdempotencyKey, items)
	if err != nil {
		return domain.Order{}, err
	}

	// Add shipping info if provided. The shipping snapshot is part of the order:
	// it is persisted with it and published in the order-paid event, so the
	// shipping domain can create the shipment when the order is paid.
	if req.ShippingQuoteID != "" && req.ShippingMethodID != "" {
		order.WithShipping(domain.ShippingSnapshot{
			QuoteID:     req.ShippingQuoteID,
			MethodID:    req.ShippingMethodID,
			MethodName:  req.ShippingMethodName,
			CostCents:   req.ShippingCostCents,
			Destination: req.DestinationSnapshot,
		})
	}

	// Reserve stock before the order exists. Keyed by the order id, so a retried
	// placement for the same order cannot double-reserve.
	if err := s.modules.ReserveStock(ctx, order.ID, lines); err != nil {
		return domain.Order{}, err
	}
	if err := s.repo.Save(ctx, order); err != nil {
		// The order was not persisted, so its reservation must not outlive it.
		// Best-effort: the ledger keyed by order id makes a later retry safe, and
		// the error from Save is what the caller needs to see.
		if releaseErr := s.modules.ReleaseStock(ctx, order.ID); releaseErr != nil {
			slog.WarnContext(ctx, "failed to release stock after save failure", "scope", metricsScope, "order_id", order.ID, "error", releaseErr)
		}
		return domain.Order{}, err
	}
	return order, nil
}

// existing resolves an idempotency key to an already-known order. A store error
// fails open to the database lookup.
func (s *Service) existing(ctx context.Context, key, fp string) (domain.Order, bool, error) {
	return s.idemCoord.Existing(ctx, key, fp)
}

// lookupByKey is the authoritative fallback: it consults the orders table
// directly, so idempotency holds even when Redis is empty or unavailable.
func (s *Service) lookupByKey(ctx context.Context, key, fp string) (domain.Order, bool, error) {
	o, err := s.repo.GetByIdempotencyKey(ctx, key)
	if errors.Is(err, domain.ErrOrderNotFound) {
		return domain.Order{}, false, nil
	}
	if err != nil {
		return domain.Order{}, false, err
	}
	if requestFingerprintOf(o.UserID, o.Items, o.ShippingQuoteID, o.ShippingMethodID, o.ShippingCostCents) != fp {
		return domain.Order{}, false, domain.ErrIdempotencyConflict
	}
	return o, true, nil
}

// reserve claims the idempotency key. It fails open: on a store error it returns
// true so processing continues, relying on the database's unique index.
func (s *Service) reserve(ctx context.Context, key, fp string) bool {
	return s.idemCoord.Reserve(ctx, key, fp)
}

// complete records the finished order under its key. A failure is logged but not
// fatal: the database remains the source of truth for replays.
func (s *Service) complete(ctx context.Context, key, fp, orderID string) {
	s.idemCoord.Complete(ctx, key, fp, orderID)
}

// release drops a reservation so a failed request can be retried.
func (s *Service) release(ctx context.Context, key, fp string) {
	s.idemCoord.Release(ctx, key, fp)
}

// requestFingerprint derives the idempotency fingerprint from the fields that
// define a placement request. Reusing a key with any of these changed is a
// conflict.
func requestFingerprint(req PlaceOrderRequest) string {
	items := make([]domain.Item, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, domain.Item{ProductID: it.ProductID, Quantity: it.Quantity})
	}
	return requestFingerprintOf(req.UserID, items, req.ShippingQuoteID, req.ShippingMethodID, req.ShippingCostCents)
}

// requestFingerprintOf fingerprints a user id and an ordered list of items, so a
// stored order and an incoming request can be compared.
func requestFingerprintOf(userID string, items []domain.Item, shippingQuoteID, shippingMethodID string, shippingCostCents int64) string {
	parts := make([]string, 0, 1+2*len(items)+3)
	parts = append(parts, userID)
	for _, it := range items {
		parts = append(parts, it.ProductID, strconv.Itoa(it.Quantity))
	}
	parts = append(parts, shippingQuoteID, shippingMethodID, strconv.FormatInt(shippingCostCents, 10))
	return idempotency.Fingerprint(parts...)
}
