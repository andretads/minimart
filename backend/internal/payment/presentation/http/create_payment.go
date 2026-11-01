package http

import (
	"encoding/json"
	"errors"
	"minimart/internal/payment/application"
	"minimart/pkg/idempotency"
	"minimart/pkg/response"
	"net/http"
)

const paymentRetryAfterSeconds = "5"

// CreatePayment charges an order, idempotently.
//
//	@Summary		Create a payment
//	@Description	Creates and processes a payment. The Idempotency-Key header is required; a retry with the same key and parameters returns the original payment, while the same key with different parameters is a conflict. A 201 includes a Location header for the created payment; a 409 while the first attempt is still processing includes Retry-After. Clients that lose the response can recover the same payment with GET /v1/payments/by-key/{key}.
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
		if errors.Is(err, application.ErrPaymentInProgress) {
			w.Header().Set("Retry-After", paymentRetryAfterSeconds)
		}
		respondDomainError(w, err)
		return
	}
	w.Header().Set("Location", "/v1/payments/"+p.ID)
	response.WriteJSON(w, http.StatusCreated, response.Created(toPaymentResponse(p)))
}
