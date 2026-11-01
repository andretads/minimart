package http

import (
	"encoding/json"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/pkg/auth"
	"minimart/pkg/idempotency"
	"minimart/pkg/response"
	"net/http"
)

// idempotency.Header carries the caller's idempotency key.

// CreateQuote handles POST /v1/shipping/quotes
//
//	@Summary		Create a shipping quote
//	@Description	Calculate a shipping quote for a customer address and method. The Idempotency-Key header is required: a retry with the same key and parameters returns the original quote.
//	@Tags			shipping
//	@Accept			json
//	@Produce		json
//	@Param			Idempotency-Key	header		string	true	"Unique key making the request safe to retry"
//	@Param			quote	body		CreateQuoteRequest	true	"Shipping quote"
//	@Success		201		{object}	response.Response{data=ShippingQuoteResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/quotes [post]
func (h *Handlers) CreateQuote(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get(idempotency.Header)
	if key == "" {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "cabeçalho Idempotency-Key é obrigatório"))
		return
	}

	var req CreateQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondDomainError(w, domain.ErrInvalidInput)
		return
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	if h.customer == nil {
		respondDomainError(w, domain.ErrInvalidInput)
		return
	}
	customer, err := h.customer.EnsureCustomer(r.Context(), userID)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	quote, err := h.svc.CreateQuote(r.Context(), application.CreateQuoteRequest{
		CustomerID:     customer.ID,
		MethodID:       req.MethodID,
		AddressID:      req.AddressID,
		SubtotalCents:  req.SubtotalCents,
		IdempotencyKey: key,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, response.Created(toQuoteResponse(quote)))
}
