// Package message holds the event payloads published across domains. An event is
// a published contract, like an HTTP response body: keep it small and stable, and
// change it only in backward-compatible ways.
package message

// OrderPlaced is emitted after an order is successfully saved. Consumers in other
// domains react to it without the order domain knowing they exist.
type OrderPlaced struct {
	OrderID    string `json:"order_id"`
	UserID     string `json:"user_id"`
	CustomerID string `json:"customer_id,omitempty"`
	TotalCents int64  `json:"total_cents"`
	ItemCount  int    `json:"item_count"`
}

// OrderCancelled is emitted after an order is successfully cancelled.
type OrderCancelled struct {
	OrderID    string `json:"order_id"`
	UserID     string `json:"user_id"`
	CustomerID string `json:"customer_id,omitempty"`
	TotalCents int64  `json:"total_cents"`
	ItemCount  int    `json:"item_count"`
}

// OrderStockRelease is emitted when an order reaches a state that will not
// consume its stock reservation (cancelled or payment failed). The catalog domain
// consumes it to restore the held stock; the order id is all it needs, because
// the reservation is keyed by order.
type OrderStockRelease struct {
	OrderID string `json:"order_id"`
}

// OrderPaid is emitted after an order is successfully paid and includes
// shipping information for the Shipping domain to create a shipment.
type OrderPaid struct {
	OrderID             string              `json:"order_id"`
	UserID              string              `json:"user_id"`
	CustomerID          string              `json:"customer_id,omitempty"`
	TotalCents          int64               `json:"total_cents"`
	ShippingQuoteID     string              `json:"shipping_quote_id"`
	ShippingMethodID    string              `json:"shipping_method_id"`
	DestinationSnapshot DestinationSnapshot `json:"destination_snapshot"`
}

// DestinationSnapshot is an immutable copy of the delivery address.
type DestinationSnapshot struct {
	Label    string `json:"label"`
	FullName string `json:"full_name"`
	Street   string `json:"street"`
	City     string `json:"city"`
	State    string `json:"state"`
	Zip      string `json:"zip"`
	Country  string `json:"country"`
	Phone    string `json:"phone,omitempty"`
}

// PaymentInitiated is emitted when a payment is created and is pending with the
// provider.
type PaymentInitiated struct {
	PaymentID   string `json:"payment_id"`
	OrderID     string `json:"order_id"`
	UserID      string `json:"user_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
}

// PaymentSucceeded is emitted after the provider authorizes and captures a
// payment.
type PaymentSucceeded struct {
	PaymentID   string `json:"payment_id"`
	OrderID     string `json:"order_id"`
	UserID      string `json:"user_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	ProviderRef string `json:"provider_ref"`
}

// PaymentFailed is emitted after the provider explicitly declines a payment.
type PaymentFailed struct {
	PaymentID   string `json:"payment_id"`
	OrderID     string `json:"order_id"`
	UserID      string `json:"user_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	Reason      string `json:"reason"`
}

// PaymentUnknown is emitted when a payment's outcome could not be determined (a
// provider timeout or technical error); it must be reconciled later.
type PaymentUnknown struct {
	PaymentID   string `json:"payment_id"`
	OrderID     string `json:"order_id"`
	UserID      string `json:"user_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	Reason      string `json:"reason"`
}

// PaymentRefunded is emitted after the provider approves reversing a
// captured payment's charge, either because the order was cancelled while the
// charge was in flight or after the payment had already succeeded.
type PaymentRefunded struct {
	PaymentID   string `json:"payment_id"`
	OrderID     string `json:"order_id"`
	UserID      string `json:"user_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	ProviderRef string `json:"provider_ref"`
	Reason      string `json:"reason"`
}

// ShipmentCreated is emitted when a shipment is created.
type ShipmentCreated struct {
	ShipmentID          string              `json:"shipment_id"`
	OrderID             string              `json:"order_id"`
	UserID              string              `json:"user_id"`
	MethodID            string              `json:"method_id"`
	ShippingCostCents   int64               `json:"shipping_cost_cents"`
	DestinationSnapshot DestinationSnapshot `json:"destination_snapshot"`
	CreatedAt           string              `json:"created_at"`
}

// ShipmentShipped is emitted when a shipment is dispatched.
type ShipmentShipped struct {
	ShipmentID   string `json:"shipment_id"`
	OrderID      string `json:"order_id"`
	UserID       string `json:"user_id"`
	TrackingCode string `json:"tracking_code"`
	Carrier      string `json:"carrier"`
	ShippedAt    string `json:"shipped_at"`
}

// ShipmentDelivered is emitted when a shipment is delivered.
type ShipmentDelivered struct {
	ShipmentID  string `json:"shipment_id"`
	OrderID     string `json:"order_id"`
	UserID      string `json:"user_id"`
	DeliveredAt string `json:"delivered_at"`
}

// ReviewCreated is emitted when a customer submits a product review.
// The review-moderation consumer group processes it for AI moderation.
type ReviewCreated struct {
	ReviewID  string `json:"review_id"`
	CreatedAt string `json:"created_at"` // RFC3339Nano
}

// ReviewModerated is emitted when a review is moderated.
type ReviewModerated struct {
	ReviewID  string `json:"review_id"`
	ProductID string `json:"product_id"`
	Decision  string `json:"decision"`
	Rating    int    `json:"rating"`
}
