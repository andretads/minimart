package domain

import (
	"minimart/pkg/message"
	"time"
)

// Item is one line of an order. Name and UnitPriceCents are a snapshot captured
// when the order is placed, so later catalog price or name changes never rewrite
// past orders.
type Item struct {
	ProductID      string
	Name           string
	UnitPriceCents int64
	Quantity       int
	LineTotalCents int64
}

// DestinationSnapshot is an immutable copy of the delivery address at the time
// of order placement, so historical orders don't depend on mutable customer
// addresses. Its JSON tags are the wire format: it is stored as JSONB on the
// order and published verbatim in the order-paid event. It is the shared
// cross-domain value type: the event payloads and the order/shipping contract
// clients alias it.
type DestinationSnapshot = message.DestinationSnapshot

// Status is an order's lifecycle state.
type Status string

const (
	// StatusPlaced is the state of a freshly placed order.
	StatusPlaced Status = "placed"
	// StatusPaid is the state of an order whose payment succeeded.
	StatusPaid Status = "paid"
	// StatusPaymentFailed is the state of an order whose payment was declined.
	StatusPaymentFailed Status = "payment_failed"
	// StatusPaymentUnknown is the state of an order whose payment outcome could
	// not be determined and must be reconciled later.
	StatusPaymentUnknown Status = "payment_unknown"
	// StatusCancelled is the state of an order that has been cancelled.
	StatusCancelled Status = "cancelled"
)

// AutoCancelAfter is how long a placed order may remain unpaid before the worker
// cancels it automatically and releases its reserved stock.
const AutoCancelAfter = 5 * time.Minute

// Order is a placed order. Like Product, it is valid by construction: NewOrder is
// the only way to make one. CancelledAt is nil until the order is cancelled.
// IdempotencyKey is the caller-supplied key that made the placement safe to
// retry; it is unique across orders.
//
// The shipping fields are the snapshot taken at placement: ShippingQuoteID and
// ShippingMethodID identify the quote the order was placed with,
// ShippingMethodName is the method's name at that moment, ShippingCostCents is
// the shipping cost folded into TotalCents, and DestinationSnapshot is the
// delivery address. They are empty on an order placed without shipping.
type Order struct {
	ID                  string
	UserID              string
	CustomerID          string
	IdempotencyKey      string
	Items               []Item
	TotalCents          int64
	Status              Status
	CancelledAt         *time.Time
	CreatedAt           time.Time
	ShippingQuoteID     string
	ShippingMethodID    string
	ShippingMethodName  string
	ShippingCostCents   int64
	DestinationSnapshot DestinationSnapshot

	// events are the domain events raised since the order was created or loaded.
	// The repository persists them to the outbox in the same transaction as the
	// order, then they are discarded with the value.
	events []Event
}

// AutoCancelAt returns the time at which a placed order becomes eligible for
// automatic cancellation. It is nil for orders that are not placed.
func (o Order) AutoCancelAt() *time.Time {
	if o.Status != StatusPlaced {
		return nil
	}
	t := o.CreatedAt.Add(AutoCancelAfter)
	return &t
}

// Events returns the domain events raised since the order was created or loaded.
func (o Order) Events() []Event { return o.events }

// NewOrder validates the lines, computes each LineTotalCents and the order
// TotalCents, and stamps CreatedAt. Callers pass items whose UnitPriceCents were
// already resolved from an authoritative source (the catalog) - never the client.
// idempotencyKey is the caller's key for this placement.
func NewOrder(id, userID, customerID, idempotencyKey string, items []Item) (Order, error) {
	if len(items) == 0 {
		return Order{}, ErrNoItems
	}
	var total int64
	for i := range items {
		if items[i].Quantity <= 0 {
			return Order{}, ErrInvalidQuantity
		}
		items[i].LineTotalCents = items[i].UnitPriceCents * int64(items[i].Quantity)
		total += items[i].LineTotalCents
	}
	return Order{
		ID:             id,
		UserID:         userID,
		CustomerID:     customerID,
		IdempotencyKey: idempotencyKey,
		Items:          items,
		TotalCents:     total,
		Status:         StatusPlaced,
		CreatedAt:      time.Now().UTC(),
		events: []Event{OrderPlacedEvent{
			OrderID:    id,
			UserID:     userID,
			CustomerID: customerID,
			TotalCents: total,
			ItemCount:  len(items),
		}},
	}, nil
}

// ShippingSnapshot is the placement-time shipping info attached to an order
// by WithShipping: the quote and method the order was placed with, their cost,
// and the delivery address.
type ShippingSnapshot struct {
	QuoteID     string
	MethodID    string
	MethodName  string
	CostCents   int64
	Destination DestinationSnapshot
}

// WithShipping attaches the shipping snapshot to the order and folds the
// shipping cost into TotalCents. Callers skip it for orders placed without
// shipping, which is why those orders carry an empty snapshot.
func (o *Order) WithShipping(s ShippingSnapshot) {
	o.ShippingQuoteID = s.QuoteID
	o.ShippingMethodID = s.MethodID
	o.ShippingMethodName = s.MethodName
	o.ShippingCostCents = s.CostCents
	o.DestinationSnapshot = s.Destination
	o.TotalCents += s.CostCents
}

// Cancel returns a copy of the order marked cancelled and raises the events that
// announce it: OrderCancelled plus OrderStockRelease, so the catalog releases the
// stock the order was holding. Cancelling an already-cancelled order reports
// ErrOrderAlreadyCancelled; cancelling a paid order reports ErrOrderNotCancellable
// (reversing a paid order is a refund/return, not a cancellation).
func (o Order) Cancel() (Order, error) {
	if o.Status == StatusCancelled {
		return Order{}, ErrOrderAlreadyCancelled
	}
	if o.Status == StatusPaid {
		return Order{}, ErrOrderNotCancellable
	}
	now := time.Now().UTC()
	o.Status = StatusCancelled
	o.CancelledAt = &now
	o.events = append(o.events,
		OrderCancelledEvent{
			OrderID:    o.ID,
			UserID:     o.UserID,
			CustomerID: o.CustomerID,
			TotalCents: o.TotalCents,
			ItemCount:  len(o.Items),
		},
		OrderStockReleaseEvent{OrderID: o.ID},
	)
	return o, nil
}

// MarkPaid returns a copy of the order marked paid. It is a no-op - returning the
// order unchanged and recording no event - when the order is already paid, was
// cancelled (a late payment event must not resurrect a cancelled order), or its
// payment already failed (a failure is terminal for fulfilment: a successful
// retry is a new order). This keeps redelivered events idempotent. It records no
// domain event: a payment outcome is already published by the payment domain, and
// the order only needs its status.
func (o Order) MarkPaid() Order {
	if o.Status == StatusPaid || o.Status == StatusCancelled || o.Status == StatusPaymentFailed {
		return o
	}
	o.Status = StatusPaid
	return o
}

// MarkPaymentFailed returns a copy of the order marked payment_failed and raises
// OrderStockRelease, so the catalog releases the stock the order was holding. It
// is a no-op unless the order is placed or has an unresolved (unknown) payment,
// so a declined payment neither downgrades a settled paid order nor overwrites
// another failure; the no-op records no event, keeping redeliveries idempotent.
func (o Order) MarkPaymentFailed() Order {
	if o.Status != StatusPlaced && o.Status != StatusPaymentUnknown {
		return o
	}
	o.Status = StatusPaymentFailed
	o.events = append(o.events, OrderStockReleaseEvent{OrderID: o.ID})
	return o
}

// MarkPaymentUnknown returns a copy of the order marked payment_unknown. It is a
// no-op unless the order is still placed: once a payment outcome is known (paid or
// failed) or the order is cancelled, an unresolved outcome must not replace it.
// The stock reservation is held, not released. Records no event.
func (o Order) MarkPaymentUnknown() Order {
	if o.Status != StatusPlaced {
		return o
	}
	o.Status = StatusPaymentUnknown
	return o
}

// MarkPaidWithShipping returns a copy of the order marked paid and records the
// OrderPaidEvent with the order's own shipping snapshot, so the event can never
// disagree with what the order carries. It is a no-op when the order is already
// paid, cancelled, or payment failed.
func (o Order) MarkPaidWithShipping() Order {
	if o.Status == StatusPaid || o.Status == StatusCancelled || o.Status == StatusPaymentFailed {
		return o
	}
	o.Status = StatusPaid
	o.events = append(o.events, OrderPaidEvent{
		OrderID:             o.ID,
		UserID:              o.UserID,
		CustomerID:          o.CustomerID,
		TotalCents:          o.TotalCents,
		ShippingQuoteID:     o.ShippingQuoteID,
		ShippingMethodID:    o.ShippingMethodID,
		DestinationSnapshot: o.DestinationSnapshot,
	})
	return o
}
