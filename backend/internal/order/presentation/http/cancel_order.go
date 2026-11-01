package http

import (
	"minimart/pkg/response"
	"net/http"
)

// CancelOrder cancels an existing order.
//
//	@Summary		Cancel an order
//	@Description	Cancels a placed order. Cancelling an already-cancelled order is a conflict.
//	@Tags			order
//	@Produce		json
//	@Param			id	path		string	true	"Order ID"
//	@Success		200	{object}	response.Response{data=orderResponse}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		409	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/orders/{id}/cancel [post]
func (h *Handlers) CancelOrder(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.CancelOrder(r.Context(), r.PathValue("id"))
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toOrderResponse(o)))
}
