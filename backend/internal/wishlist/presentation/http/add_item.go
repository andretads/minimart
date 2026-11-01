package wishlisthttp

import (
	"encoding/json"
	"minimart/internal/wishlist/application"
	"minimart/pkg/response"
	"net/http"
)

// AddItem adds a product to the wishlist.
//
//	@Summary		Add item to wishlist
//	@Description	Adds a product to the wishlist. If the product already exists, returns an error.
//	@Tags			wishlist
//	@Accept			json
//	@Produce		json
//	@Param			request	body		addItemRequest		true	"Item to add"
//	@Success		200		{object}	response.Response{data=wishlistResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/wishlist/items [post]
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
	wishlist, err := h.svc.AddItem(r.Context(), application.AddItemRequest{
		UserID:    userID,
		ProductID: req.ProductID,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toWishlistResponse(wishlist)))
}
