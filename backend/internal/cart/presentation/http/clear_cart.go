package carthttp

import (
	"minimart/internal/cart/application"
	"minimart/pkg/response"
	"net/http"
)

// ClearCart removes all items from the cart.
//
//	@Summary		Clear cart
//	@Description	Removes all items from the cart. The cart itself is not deleted.
//	@Tags			cart
//	@Produce		json
//	@Success		204		{object}	response.Response
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/cart [delete]
func (h *Handlers) ClearCart(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	_, err := h.svc.ClearCart(r.Context(), application.ClearCartRequest{UserID: userID})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
