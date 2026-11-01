package http

import (
	"encoding/json"
	"minimart/internal/order/application"
	"minimart/pkg/idempotency"
	"minimart/pkg/response"
	"net/http"
)

// idempotency.Header carries the caller's idempotency key.

// PlaceOrder places an order for one or more products.
//
//	@Summary		Place an order
//	@Description	Places an order. Prices are resolved server-side; the client only sends product IDs and quantities. The Idempotency-Key header is required: a retry with the same key and parameters returns the original order.
//	@Tags			order
//	@Accept			json
//	@Produce		json
//	@Param			Idempotency-Key	header		string	true	"Unique key making the request safe to retry"
//	@Param			order	body		placeOrderRequest	true	"Order to place"
//	@Success		201		{object}	response.Response{data=orderResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/orders [post]
func (h *Handlers) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get(idempotency.Header)
	if key == "" {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "cabeçalho Idempotency-Key é obrigatório"))
		return
	}

	var req placeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo JSON inválido"))
		return
	}
	items := make([]application.PlaceOrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, application.PlaceOrderItem{ProductID: it.ProductID, Quantity: it.Quantity})
	}
	o, err := h.svc.PlaceOrder(r.Context(), application.PlaceOrderRequest{
		UserID:         req.UserID,
		IdempotencyKey: key,
		Items:          items,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, response.Created(toOrderResponse(o)))
}
