package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"minimart/internal/order/domain"
	"minimart/internal/order/infrastructure/postgres/reader"
	"minimart/internal/order/infrastructure/postgres/writer"
	"minimart/pkg/outbox"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventPublisher writes a domain event into the transactional outbox within the
// caller's transaction, so the event commits atomically with the order.
// *outbox.Publisher is the production implementation.
type EventPublisher interface {
	Send(ctx context.Context, msg outbox.Message, tx pgx.Tx) error
}

type OrderRepository struct {
	writePool *pgxpool.Pool
	reader    *reader.Queries
	writer    *writer.Queries
	events    EventPublisher
}

var _ domain.Repository = (*OrderRepository)(nil)

func NewOrderRepository(readPool, writePool *pgxpool.Pool, events EventPublisher) *OrderRepository {
	return &OrderRepository{
		writePool: writePool,
		reader:    reader.New(readPool),
		writer:    writer.New(writePool),
		events:    events,
	}
}

// Save writes the order, all its items, and the order's domain events in one
// transaction. If anything fails, the deferred Rollback undoes the whole thing -
// so there is never an order without its outbox event, nor an event without its
// order.
//
// The insert uses ON CONFLICT DO NOTHING against the idempotency-key index: when
// the key already belongs to another order, no row is written and
// domain.ErrIdempotencyConflict is reported so the caller can replay the existing
// order instead of creating a second one. An empty key never conflicts.
func (r *OrderRepository) Save(ctx context.Context, o domain.Order) error {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	q := r.writer.WithTx(tx)
	custID := (*string)(nil)
	if o.CustomerID != "" {
		v := o.CustomerID
		custID = &v
	}
	destinationJSON, err := marshalDestinationSnapshot(o.DestinationSnapshot)
	if err != nil {
		return err
	}
	rows, err := q.CreateOrder(ctx, writer.CreateOrderParams{
		ID:                  o.ID,
		UserID:              o.UserID,
		CustomerID:          custID,
		TotalCents:          o.TotalCents,
		IdempotencyKey:      o.IdempotencyKey,
		ShippingQuoteID:     o.ShippingQuoteID,
		ShippingMethodID:    o.ShippingMethodID,
		ShippingMethodName:  o.ShippingMethodName,
		ShippingCostCents:   o.ShippingCostCents,
		DestinationSnapshot: destinationJSON,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrIdempotencyConflict
	}
	for _, it := range o.Items {
		// The quantity column is a 4-byte integer; reject values that would
		// overflow it rather than silently truncating.
		if it.Quantity > math.MaxInt32 {
			return fmt.Errorf("item quantity %d exceeds int32 range", it.Quantity)
		}
		if err := q.CreateOrderItem(ctx, writer.CreateOrderItemParams{
			OrderID:        o.ID,
			ProductID:      it.ProductID,
			Name:           it.Name,
			UnitPriceCents: it.UnitPriceCents,
			Quantity:       int32(it.Quantity), //nolint:gosec // bounded by the range check above
			LineTotalCents: it.LineTotalCents,
		}); err != nil {
			return err
		}
	}
	if err := r.appendEvents(ctx, tx, o.Events()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (domain.Order, error) {
	row, err := r.reader.GetOrder(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, domain.ErrOrderNotFound
		}
		return domain.Order{}, err
	}
	return r.assemble(ctx, row)
}

// GetByIdempotencyKey returns the order created under an idempotency key, or
// ErrOrderNotFound when none carries it. This is the authoritative idempotency
// lookup: it holds even if the Redis cache is empty.
func (r *OrderRepository) GetByIdempotencyKey(ctx context.Context, key string) (domain.Order, error) {
	row, err := r.reader.GetOrderByIdempotencyKey(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, domain.ErrOrderNotFound
		}
		return domain.Order{}, err
	}
	return r.assemble(ctx, row)
}

// ListByUserID returns one page of a user's orders, newest first, with each
// order's items loaded in a single follow-up query.
func (r *OrderRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]domain.Order, error) {
	rows, err := r.reader.ListOrdersByUserID(ctx, reader.ListOrdersByUserIDParams{
		UserID: userID,
		Limit:  int32(limit),  //nolint:gosec // limit validated <= 100
		Offset: int32(offset), //nolint:gosec // offset from validated page/page_size
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []domain.Order{}, nil
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	itemRows, err := r.reader.ListOrderItemsByOrderIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	itemsByOrder := make(map[string][]domain.Item, len(rows))
	for _, it := range itemRows {
		itemsByOrder[it.OrderID] = append(itemsByOrder[it.OrderID], domain.Item{
			ProductID:      it.ProductID,
			Name:           it.Name,
			UnitPriceCents: it.UnitPriceCents,
			Quantity:       int(it.Quantity),
			LineTotalCents: it.LineTotalCents,
		})
	}

	orders := make([]domain.Order, 0, len(rows))
	for _, row := range rows {
		dest, err := unmarshalDestinationSnapshot(row.DestinationSnapshot)
		if err != nil {
			return nil, err
		}
		customerID := ""
		if row.CustomerID != nil {
			customerID = *row.CustomerID
		}
		orders = append(orders, domain.Order{
			ID:                  row.ID,
			UserID:              row.UserID,
			CustomerID:          customerID,
			IdempotencyKey:      row.IdempotencyKey,
			TotalCents:          row.TotalCents,
			Items:               itemsByOrder[row.ID],
			Status:              domain.Status(row.Status),
			CancelledAt:         row.CancelledAt,
			CreatedAt:           row.CreatedAt,
			ShippingQuoteID:     row.ShippingQuoteID,
			ShippingMethodID:    row.ShippingMethodID,
			ShippingMethodName:  row.ShippingMethodName,
			ShippingCostCents:   row.ShippingCostCents,
			DestinationSnapshot: dest,
		})
	}
	return orders, nil
}

// CountByUserID returns the total number of orders owned by userID.
func (r *OrderRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	count, err := r.reader.CountOrdersByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// ListPlacedBefore returns placed orders created at or before before, oldest first,
// with each order's items loaded in a single follow-up query.
func (r *OrderRepository) ListPlacedBefore(ctx context.Context, before time.Time, limit int) ([]domain.Order, error) {
	rows, err := r.reader.ListPlacedOrdersBefore(ctx, reader.ListPlacedOrdersBeforeParams{
		CreatedAt: before,
		Limit:     int32(limit), //nolint:gosec // limit validated <= 1000
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []domain.Order{}, nil
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	itemRows, err := r.reader.ListOrderItemsByOrderIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	itemsByOrder := make(map[string][]domain.Item, len(rows))
	for _, it := range itemRows {
		itemsByOrder[it.OrderID] = append(itemsByOrder[it.OrderID], domain.Item{
			ProductID:      it.ProductID,
			Name:           it.Name,
			UnitPriceCents: it.UnitPriceCents,
			Quantity:       int(it.Quantity),
			LineTotalCents: it.LineTotalCents,
		})
	}

	orders := make([]domain.Order, 0, len(rows))
	for _, row := range rows {
		dest, err := unmarshalDestinationSnapshot(row.DestinationSnapshot)
		if err != nil {
			return nil, err
		}
		customerID := ""
		if row.CustomerID != nil {
			customerID = *row.CustomerID
		}
		orders = append(orders, domain.Order{
			ID:                  row.ID,
			UserID:              row.UserID,
			CustomerID:          customerID,
			IdempotencyKey:      row.IdempotencyKey,
			TotalCents:          row.TotalCents,
			Items:               itemsByOrder[row.ID],
			Status:              domain.Status(row.Status),
			CancelledAt:         row.CancelledAt,
			CreatedAt:           row.CreatedAt,
			ShippingQuoteID:     row.ShippingQuoteID,
			ShippingMethodID:    row.ShippingMethodID,
			ShippingMethodName:  row.ShippingMethodName,
			ShippingCostCents:   row.ShippingCostCents,
			DestinationSnapshot: dest,
		})
	}
	return orders, nil
}

// assemble loads an order's items and builds the domain value from the row's
// scalar fields.
func (r *OrderRepository) assemble(ctx context.Context, row reader.Order) (domain.Order, error) {
	items, err := r.reader.ListOrderItems(ctx, row.ID)
	if err != nil {
		return domain.Order{}, err
	}
	domItems := make([]domain.Item, 0, len(items))
	for _, it := range items {
		domItems = append(domItems, domain.Item{
			ProductID:      it.ProductID,
			Name:           it.Name,
			UnitPriceCents: it.UnitPriceCents,
			Quantity:       int(it.Quantity),
			LineTotalCents: it.LineTotalCents,
		})
	}
	dest, err := unmarshalDestinationSnapshot(row.DestinationSnapshot)
	if err != nil {
		return domain.Order{}, err
	}
	customerID := ""
	if row.CustomerID != nil {
		customerID = *row.CustomerID
	}
	return domain.Order{
		ID:                  row.ID,
		UserID:              row.UserID,
		CustomerID:          customerID,
		IdempotencyKey:      row.IdempotencyKey,
		TotalCents:          row.TotalCents,
		Items:               domItems,
		Status:              domain.Status(row.Status),
		CancelledAt:         row.CancelledAt,
		CreatedAt:           row.CreatedAt,
		ShippingQuoteID:     row.ShippingQuoteID,
		ShippingMethodID:    row.ShippingMethodID,
		ShippingMethodName:  row.ShippingMethodName,
		ShippingCostCents:   row.ShippingCostCents,
		DestinationSnapshot: dest,
	}, nil
}

// marshalDestinationSnapshot encodes the destination snapshot for the JSONB
// column. An order placed without shipping stores NULL.
func marshalDestinationSnapshot(dest domain.DestinationSnapshot) ([]byte, error) {
	if dest == (domain.DestinationSnapshot{}) {
		return nil, nil
	}
	raw, err := json.Marshal(dest)
	if err != nil {
		return nil, fmt.Errorf("marshal destination snapshot: %w", err)
	}
	return raw, nil
}

// unmarshalDestinationSnapshot decodes the JSONB column back into the snapshot.
// A NULL column (an order placed without shipping) yields the zero snapshot.
func unmarshalDestinationSnapshot(raw []byte) (domain.DestinationSnapshot, error) {
	if len(raw) == 0 {
		return domain.DestinationSnapshot{}, nil
	}
	var dest domain.DestinationSnapshot
	if err := json.Unmarshal(raw, &dest); err != nil {
		return domain.DestinationSnapshot{}, fmt.Errorf("unmarshal destination snapshot: %w", err)
	}
	return dest, nil
}

// Update persists a change to an existing order's state together with the order's
// domain events, in one transaction. The status write is a compare-and-set against
// expected: if no row matches, a follow-up probe on the write pool distinguishes a
// missing order (domain.ErrOrderNotFound) from a lost race
// (domain.ErrOrderStateChanged).
func (r *OrderRepository) Update(ctx context.Context, o domain.Order, expected domain.Status) error {
	tx, err := r.writePool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.writer.WithTx(tx)
	rows, err := q.UpdateOrderStatus(ctx, writer.UpdateOrderStatusParams{
		ID:             o.ID,
		NewStatus:      string(o.Status),
		CancelledAt:    o.CancelledAt,
		ExpectedStatus: string(expected),
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		// Probe on the same (primary) pool so replica lag cannot make an existing
		// order look missing.
		exists, err := q.OrderExists(ctx, o.ID)
		if err != nil {
			return err
		}
		if !exists {
			return domain.ErrOrderNotFound
		}
		return domain.ErrOrderStateChanged
	}
	if err := r.appendEvents(ctx, tx, o.Events()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// appendEvents writes each domain event to the outbox inside tx. Writing through
// the shared outbox store (which participates in the same tx) is what makes the
// state change and its events atomic.
func (r *OrderRepository) appendEvents(ctx context.Context, tx pgx.Tx, events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}
	if r.events == nil {
		return errors.New("order repository: no event publisher configured")
	}
	for _, ev := range events {
		msg, err := outboxMessage(ev)
		if err != nil {
			return err
		}
		if err := r.events.Send(ctx, msg, tx); err != nil {
			return fmt.Errorf("enqueue order event: %w", err)
		}
	}
	return nil
}
