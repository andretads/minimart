package http

import (
	"minimart/internal/shipping/domain"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

// GetQuote handles GET /v1/shipping/quotes/{id}
//
//	@Summary		Get a shipping quote
//	@Description	Returns a shipping quote by its ID.
//	@Tags			shipping
//	@Produce		json
//	@Param			id	path		string	true	"Shipping quote ID"
//	@Success		200	{object}	response.Response{data=ShippingQuoteResponse}
//	@Failure		401	{object}	response.Response{error=response.ErrorBody}
//	@Failure		403	{object}	response.Response{error=response.ErrorBody}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/quotes/{id} [get]
func (h *Handlers) GetQuote(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	id := r.PathValue("id")
	quote, err := h.svc.GetQuote(r.Context(), id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if quote.CustomerID != userID {
		respondDomainError(w, domain.ErrOwnershipMismatch)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toQuoteResponse(quote)))
}
