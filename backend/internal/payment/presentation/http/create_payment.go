package http

import (
	"encoding/json"
	"minimart/internal/payment/application"
	"minimart/pkg/idempotency"
	"minimart/pkg/response"
	"net/http"
)

// CreatePayment charges an order, idempotently.
//
//	@Summary		Create a payment
//	@Description	Creates and processes a payment. The Idempotency-Key header is required; a retry with the same key and parameters returns the original payment, while the same key with different parameters is a conflict.
//	@Tags			payment
//	@Accept			json
//	@Produce		json
//	@Param			Idempotency-Key	header		string					true	"Unique key making the request safe to retry"
//	@Param			payment			body		createPaymentRequest	true	"Payment to create"
//	@Success		201				{object}	response.Response{data=paymentResponse}
//	@Failure		400				{object}	response.Response{error=response.ErrorBody}
//	@Failure		409				{object}	response.Response{error=response.ErrorBody}
//	@Failure		500				{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/payments [post]
func (h *Handlers) CreatePayment(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get(idempotency.Header)
	if key == "" {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "cabeçalho Idempotency-Key é obrigatório"))
		return
	}

	var req createPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo JSON inválido"))
		return
	}

	p, err := h.svc.CreatePayment(r.Context(), application.CreatePaymentRequest{
		OrderID:        req.OrderID,
		UserID:         req.UserID,
		IdempotencyKey: key,
		AmountCents:    req.AmountCents,
		Currency:       req.Currency,
		Method:         req.Method,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, response.Created(toPaymentResponse(p)))
}
