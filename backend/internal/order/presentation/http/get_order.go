package http

import (
	"minimart/pkg/response"
	"net/http"
)

// GetOrder returns a single order by ID.
//
//	@Summary		Get an order
//	@Description	Returns an order by its ID.
//	@Tags			order
//	@Produce		json
//	@Param			id	path		string	true	"Order ID"
//	@Success		200	{object}	response.Response{data=orderResponse}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/orders/{id} [get]
func (h *Handlers) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	o, err := h.svc.GetOrder(r.Context(), id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toOrderResponse(o)))
}
