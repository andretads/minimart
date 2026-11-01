package carthttp

import (
	"encoding/json"
	"minimart/internal/cart/application"
	"minimart/pkg/response"
	"net/http"
)

// AddItem adds an item to the cart or increments quantity if already present.
//
//	@Summary		Add item to cart
//	@Description	Adds a product to the cart. If the product already exists, its quantity is incremented.
//	@Tags			cart
//	@Accept			json
//	@Produce		json
//	@Param			request	body		addItemRequest		true	"Item to add"
//	@Success		200		{object}	response.Response{data=cartResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/cart/items [post]
func (h *Handlers) AddItem(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	var req addItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo da requisição inválido"))
		return
	}
	cart, err := h.svc.AddItem(r.Context(), application.AddItemRequest{
		UserID:    userID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toCartResponse(cart)))
}
