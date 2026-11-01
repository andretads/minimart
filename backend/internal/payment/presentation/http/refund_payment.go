package http

import (
	"minimart/pkg/response"
	"net/http"
)

// RefundPayment refunds a captured payment.
//
//	@Summary		Refund a payment
//	@Description	Refunds a succeeded payment whose order was cancelled. Refunding an already-refunded payment returns it unchanged; a payment whose order is not cancelled is a conflict.
//	@Tags			payment
//	@Produce		json
//	@Param			id	path		string	true	"Payment ID"
//	@Success		200	{object}	response.Response{data=paymentResponse}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		409	{object}	response.Response{error=response.ErrorBody}
//	@Failure		502	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/payments/{id}/refund [post]
func (h *Handlers) RefundPayment(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.RefundPayment(r.Context(), r.PathValue("id"))
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toPaymentResponse(p)))
}
