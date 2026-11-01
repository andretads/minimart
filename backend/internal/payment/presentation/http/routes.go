package http

import "minimart/pkg/routes"

// Register wires the payment routes onto the shared mux.
func (h *Handlers) Register(mux routes.Registrar) {
	mux.HandleFunc("POST /v1/payments", h.CreatePayment)
	mux.HandleFunc("GET /v1/payments/by-key/{key}", h.GetPaymentByKey)
	mux.HandleFunc("GET /v1/payments/{id}", h.GetPayment)
	mux.HandleFunc("POST /v1/payments/{id}/refund", h.RefundPayment)
}
