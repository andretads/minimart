package wishlisthttp

import (
	"minimart/internal/wishlist/application"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

func getUserID(r *http.Request) string {
	userID, _ := auth.UserIDFromContext(r.Context())
	return userID
}

// GetWishlist returns the authenticated user's wishlist, creating one if none exists.
//
//	@Summary		Get wishlist
//	@Description	Returns the authenticated user's wishlist. Creates a new wishlist if none exists.
//	@Tags			wishlist
//	@Produce		json
//	@Success		200		{object}	response.Response{data=wishlistResponse}
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/wishlist [get]
func (h *Handlers) GetWishlist(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	wishlist, err := h.svc.GetWishlist(r.Context(), application.GetWishlistRequest{UserID: userID})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toWishlistResponse(wishlist)))
}
