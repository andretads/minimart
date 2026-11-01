package carthttp

import (
	"encoding/json"
	"minimart/internal/cart/application"
	"minimart/pkg/response"
	"net/http"
)

// UpdateItem sets the quantity for a specific product in the cart.
//
//	@Summary		Update item quantity
//	@Description	Sets the quantity for a product in the cart. Quantity must be >= 1.
//	@Tags			cart
//	@Accept			json
//	@Produce		json
//	@Param			product_id	path	string				true	"Product ID"
//	@Param			request	body		updateItemRequest		true	"New quantity"
//	@Success		200		{object}	response.Response{data=cartResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/cart/items/{product_id} [patch]
func (h *Handlers) UpdateItem(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	productID := r.PathValue("product_id")
	var req updateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo da requisição inválido"))
		return
	}
	cart, err := h.svc.UpdateItemQuantity(r.Context(), application.UpdateItemQuantityRequest{
		UserID:    userID,
		ProductID: productID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toCartResponse(cart)))
}
