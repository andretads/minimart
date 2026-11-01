package http

import "minimart/pkg/routes"

func (h *Handlers) Register(mux routes.Registrar) {
	mux.HandleFunc("POST /v1/addresses", h.CreateAddress)
	mux.HandleFunc("GET /v1/addresses", h.ListAddresses)
	mux.HandleFunc("GET /v1/addresses/{id}", h.GetAddress)
	mux.HandleFunc("PATCH /v1/addresses/{id}", h.UpdateAddress)
	mux.HandleFunc("DELETE /v1/addresses/{id}", h.DeleteAddress)
	mux.HandleFunc("POST /v1/addresses/{id}/default", h.SetDefaultAddress)
}
