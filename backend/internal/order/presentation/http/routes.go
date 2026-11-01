package http

import "minimart/pkg/routes"

func (h *Handlers) Register(mux routes.Registrar) {
	mux.HandleFunc("POST /v1/orders", h.PlaceOrder)
	mux.HandleFunc("GET /v1/orders", h.ListOrders)
	mux.HandleFunc("GET /v1/orders/{id}", h.GetOrder)
	mux.HandleFunc("POST /v1/orders/{id}/cancel", h.CancelOrder)
}
