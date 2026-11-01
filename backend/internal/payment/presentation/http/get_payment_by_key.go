package http

import (
	"minimart/internal/payment/domain"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

// GetPaymentByKey returns the payment created for an idempotency key, so a
// client whose response was lost can resume the same operation.
//
//	@Summary		Get a payment by idempotency key
//	@Description	Returns the payment created for a previously used Idempotency-Key. The authenticated user must own the payment.
//	@Tags			payment
//	@Produce		json
//	@Param			key	path		string	true	"Idempotency-Key used when the payment was created"
//	@Success		200	{object}	response.Response{data=paymentResponse}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/payments/by-key/{key} [get]
func (h *Handlers) GetPaymentByKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		respondDomainError(w, domain.ErrPaymentNotFound)
		return
	}
	p, err := h.svc.GetPaymentByKey(r.Context(), r.PathValue("key"), userID)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toPaymentResponse(p)))
}
