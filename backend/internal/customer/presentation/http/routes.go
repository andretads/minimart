package http

import (
	"minimart/pkg/routes"
)

// Register wires the customer routes onto the shared mux.
func (h *Handlers) Register(mux routes.Registrar) {
	mux.HandleFunc("POST /v1/customers", h.CreateCustomer)
	mux.HandleFunc("GET /v1/customers/{id}", h.GetCustomer)
	mux.HandleFunc("GET /v1/customers", h.GetCustomerByUserID)
	mux.HandleFunc("PATCH /v1/customers/{id}", h.UpdateCustomer)
	mux.HandleFunc("POST /v1/customers/{id}/deactivate", h.DeactivateCustomer)
}
