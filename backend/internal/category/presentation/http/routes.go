package http

import "minimart/pkg/routes"

// Register wires the category routes onto the shared mux.
func (h *Handlers) Register(mux routes.Registrar) {
	mux.HandleFunc("POST /v1/categories", h.CreateCategory)
	mux.HandleFunc("GET /v1/categories", h.ListCategories)
	mux.HandleFunc("GET /v1/categories/{id}", h.GetCategory)
	mux.HandleFunc("PATCH /v1/categories/{id}", h.UpdateCategory)
	mux.HandleFunc("DELETE /v1/categories/{id}", h.DeleteCategory)
}
