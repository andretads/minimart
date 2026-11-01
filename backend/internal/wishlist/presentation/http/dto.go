package wishlisthttp

type addItemRequest struct {
	ProductID string `json:"product_id"`
} // @name AddItemRequest

type wishlistItemResponse struct {
	ProductID string `json:"product_id"`
	AddedAt   string `json:"added_at"`
} // @name WishlistItemResponse

type wishlistResponse struct {
	ID         string                 `json:"id"`
	CustomerID string                 `json:"customer_id"`
	Items      []wishlistItemResponse `json:"items"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
} // @name WishlistResponse
