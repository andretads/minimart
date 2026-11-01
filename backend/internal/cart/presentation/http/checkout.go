package carthttp

import (
	"encoding/json"
	"errors"
	"io"
	"minimart/internal/cart/application"
	"minimart/pkg/idempotency"
	"minimart/pkg/response"
	"net/http"
)

type checkoutRequest struct {
	ShippingQuoteID string `json:"shipping_quote_id"`
	AddressID       string `json:"address_id"`
}

// Checkout creates an order from the cart.
//
//	@Summary		Checkout cart
//	@Description	Creates an order from the cart items. Requires Idempotency-Key header and a shipping quote.
//	@Tags			cart
//	@Accept			json
//	@Produce		json
//	@Param			Idempotency-Key	header	string			true	"Idempotency key"
//	@Param			checkout		body	checkoutRequest	true	"Shipping quote and address"
//	@Success		201		{object}	response.Response{data=checkoutResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/cart/checkout [post]
func (h *Handlers) Checkout(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	idempotencyKey := r.Header.Get(idempotency.Header)
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo JSON inválido"))
		return
	}
	ord, err := h.svc.CheckoutCart(r.Context(), application.CheckoutCartRequest{
		UserID:          userID,
		IdempotencyKey:  idempotencyKey,
		ShippingQuoteID: req.ShippingQuoteID,
		AddressID:       req.AddressID,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, response.Created(toCheckoutResponse(ord)))
}
