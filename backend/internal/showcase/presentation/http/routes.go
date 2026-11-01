package http

import (
	"minimart/pkg/routes"
)

// Register mounts the showcase HTTP routes on the given ServeMux.
func (h *Handlers) Register(mux routes.Registrar) {
	mux.HandleFunc("GET /v1/showcase/products", h.ListProducts)
	mux.HandleFunc("GET /v1/showcase/products/{id}", h.GetProduct)
	mux.HandleFunc("GET /v1/showcase/products/{id}/reviews", h.ListProductReviews)
	mux.HandleFunc("GET /v1/showcase/categories", h.ListCategories)
}
