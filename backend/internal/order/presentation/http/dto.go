package http

import (
	"minimart/internal/order/domain"
	"time"
)

// placeOrderRequest is the body of POST /v1/orders. Note there is no price field:
// the client cannot set prices. It only names products and quantities.
type placeOrderRequest struct {
	UserID string           `json:"user_id"`
	Items  []placeOrderItem `json:"items"`
} //	@name	OrderPlaceRequest

type placeOrderItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
} //	@name	OrderItemInput

type orderResponse struct {
	ID           string         `json:"id"`
	UserID       string         `json:"user_id"`
	Items        []itemResponse `json:"items"`
	TotalCents   int64          `json:"total_cents"`
	Status       string         `json:"status"`
	CancelledAt  *string        `json:"cancelled_at,omitempty"`
	CreatedAt    string         `json:"created_at"`
	AutoCancelAt *string        `json:"auto_cancel_at,omitempty"`

	// The shipping snapshot taken at placement. It is empty on an order placed
	// without shipping.
	ShippingQuoteID     string               `json:"shipping_quote_id,omitempty"`
	ShippingMethodID    string               `json:"shipping_method_id,omitempty"`
	ShippingMethodName  string               `json:"shipping_method_name,omitempty"`
	ShippingCostCents   int64                `json:"shipping_cost_cents"`
	DestinationSnapshot *destinationSnapshot `json:"destination_snapshot,omitempty"`
} //	@name	Order

// destinationSnapshot is the delivery address captured when the order was
// placed, so a later change to the customer's address never rewrites history.
type destinationSnapshot struct {
	Label    string `json:"label"`
	FullName string `json:"full_name"`
	Street   string `json:"street"`
	City     string `json:"city"`
	State    string `json:"state"`
	Zip      string `json:"zip"`
	Country  string `json:"country"`
	Phone    string `json:"phone,omitempty"`
} //	@name	OrderDestinationSnapshot

type itemResponse struct {
	ProductID      string `json:"product_id"`
	Name           string `json:"name"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	Quantity       int    `json:"quantity"`
	LineTotalCents int64  `json:"line_total_cents"`
} //	@name	OrderItem

func toOrderResponse(o domain.Order) orderResponse {
	items := make([]itemResponse, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, itemResponse{
			ProductID:      it.ProductID,
			Name:           it.Name,
			UnitPriceCents: it.UnitPriceCents,
			Quantity:       it.Quantity,
			LineTotalCents: it.LineTotalCents,
		})
	}
	var autoCancelAt *string
	if o.Status == domain.StatusPlaced {
		value := o.CreatedAt.Add(domain.AutoCancelAfter).Format(time.RFC3339)
		autoCancelAt = &value
	}

	return orderResponse{
		ID:                  o.ID,
		UserID:              o.UserID,
		Items:               items,
		TotalCents:          o.TotalCents,
		Status:              string(o.Status),
		CancelledAt:         cancelledAtString(o.CancelledAt),
		CreatedAt:           o.CreatedAt.Format(time.RFC3339),
		AutoCancelAt:        autoCancelAt,
		ShippingQuoteID:     o.ShippingQuoteID,
		ShippingMethodID:    o.ShippingMethodID,
		ShippingMethodName:  o.ShippingMethodName,
		ShippingCostCents:   o.ShippingCostCents,
		DestinationSnapshot: toDestinationSnapshotResponse(o.DestinationSnapshot),
	}
}

// toDestinationSnapshotResponse maps the snapshot, or nil when the order was
// placed without shipping.
func toDestinationSnapshotResponse(d domain.DestinationSnapshot) *destinationSnapshot {
	if d == (domain.DestinationSnapshot{}) {
		return nil
	}
	return &destinationSnapshot{
		Label:    d.Label,
		FullName: d.FullName,
		Street:   d.Street,
		City:     d.City,
		State:    d.State,
		Zip:      d.Zip,
		Country:  d.Country,
		Phone:    d.Phone,
	}
}

func cancelledAtString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	return new(t.Format(time.RFC3339))
}
