package carthttp

import (
	"minimart/internal/cart/application"
	"minimart/pkg/response"
	"net/http"
)

// RemoveItem removes a specific product from the cart.
//
//	@Summary		Remove item from cart
//	@Description	Removes a product from the cart by product ID.
//	@Tags			cart
//	@Produce		json
//	@Param			product_id	path	string	true	"Product ID"
//	@Success		200		{object}	response.Response{data=cartResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/cart/items/{product_id} [delete]
func (h *Handlers) RemoveItem(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	productID := r.PathValue("product_id")
	cart, err := h.svc.RemoveItem(r.Context(), application.RemoveItemRequest{
		UserID:    userID,
		ProductID: productID,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toCartResponse(cart)))
}
