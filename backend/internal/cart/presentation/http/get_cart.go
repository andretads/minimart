package carthttp

import (
	"minimart/internal/cart/application"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

func getUserID(r *http.Request) string {
	userID, _ := auth.UserIDFromContext(r.Context())
	return userID
}

// GetCart returns the authenticated user's cart, creating one if none exists.
//
//	@Summary		Get cart
//	@Description	Returns the authenticated user's cart. Creates a new cart if none exists.
//	@Tags			cart
//	@Produce		json
//	@Success		200		{object}	response.Response{data=cartResponse}
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/cart [get]
func (h *Handlers) GetCart(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	cart, err := h.svc.GetOrCreateCart(r.Context(), application.GetOrCreateCartRequest{UserID: userID})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toCartResponse(cart)))
}
