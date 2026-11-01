package wishlisthttp

import (
	"minimart/internal/wishlist/application"
	"minimart/pkg/response"
	"net/http"
)

// ClearWishlist removes all items from the wishlist. The wishlist itself is not deleted.
//
//	@Summary		Clear wishlist
//	@Description	Removes all items from the wishlist. The wishlist itself is not deleted.
//	@Tags			wishlist
//	@Produce		json
//	@Success		204		{object}	response.Response
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/wishlist [delete]
func (h *Handlers) ClearWishlist(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	err := h.svc.ClearWishlist(r.Context(), application.ClearWishlistRequest{UserID: userID})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusNoContent, response.OK(nil))
}
