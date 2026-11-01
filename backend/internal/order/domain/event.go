package domain

// Event is a domain event raised by the Order aggregate. Infrastructure maps each
// concrete event to a serialized broker payload in the same transaction that
// persists the order, so events are published at least once via the outbox.
type Event interface {
	// isOrderEvent seals the interface to this package's event types.
	isOrderEvent()
}

// OrderPlacedEvent records that an order was created.
type OrderPlacedEvent struct {
	OrderID    string
	UserID     string
	CustomerID string
	TotalCents int64
	ItemCount  int
}

func (OrderPlacedEvent) isOrderEvent() {}

// OrderCancelledEvent records that an order was cancelled.
type OrderCancelledEvent struct {
	OrderID    string
	UserID     string
	CustomerID string
	TotalCents int64
	ItemCount  int
}

func (OrderCancelledEvent) isOrderEvent() {}

// OrderStockReleaseEvent records that an order has reached a state that will not
// consume its stock reservation (it was cancelled or its payment failed). The
// catalog domain consumes it to release the stock the order reserved.
type OrderStockReleaseEvent struct {
	OrderID string
}

func (OrderStockReleaseEvent) isOrderEvent() {}

// OrderPaidEvent records that an order was successfully paid and includes
// shipping information for the Shipping domain to create a shipment.
type OrderPaidEvent struct {
	OrderID             string
	UserID              string
	CustomerID          string
	TotalCents          int64
	ShippingQuoteID     string
	ShippingMethodID    string
	DestinationSnapshot DestinationSnapshot
}

func (OrderPaidEvent) isOrderEvent() {}
