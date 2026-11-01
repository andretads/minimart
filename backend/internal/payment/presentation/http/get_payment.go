package http

import (
	"minimart/pkg/response"
	"net/http"
)

// GetPayment returns a single payment by ID.
//
//	@Summary		Get a payment
//	@Description	Returns a payment by its ID.
//	@Tags			payment
//	@Produce		json
//	@Param			id	path		string	true	"Payment ID"
//	@Success		200	{object}	response.Response{data=paymentResponse}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/payments/{id} [get]
func (h *Handlers) GetPayment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := h.svc.GetPayment(r.Context(), id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toPaymentResponse(p)))
}
