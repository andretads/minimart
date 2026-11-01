package carthttp

type addItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
} // @name AddItemRequest

type updateItemRequest struct {
	Quantity int `json:"quantity"`
} // @name UpdateItemRequest

type cartItemResponse struct {
	ProductID              string `json:"product_id"`
	ProductNameSnapshot    string `json:"product_name_snapshot"`
	UnitPriceSnapshotCents int64  `json:"unit_price_snapshot_cents"`
	Quantity               int    `json:"quantity"`
	LineTotalCents         int64  `json:"line_total_cents"`
} // @name CartItemResponse

type cartResponse struct {
	ID                  string             `json:"id"`
	CustomerID          string             `json:"customer_id"`
	Status              string             `json:"status"`
	Items               []cartItemResponse `json:"items"`
	EstimatedTotalCents int64              `json:"estimated_total_cents"`
	CreatedAt           string             `json:"created_at"`
	UpdatedAt           string             `json:"updated_at"`
} // @name CartResponse

type checkoutResponse struct {
	OrderID    string `json:"order_id"`
	UserID     string `json:"user_id"`
	CustomerID string `json:"customer_id"`
	Status     string `json:"status"`
	TotalCents int64  `json:"total_cents"`
	CreatedAt  string `json:"created_at"`
} // @name CheckoutResponse
