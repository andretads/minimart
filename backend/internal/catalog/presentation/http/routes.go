package http

import "minimart/pkg/routes"

// Register wires the catalog routes onto the shared mux. Method-aware patterns
// (Go 1.22+) let the standard router dispatch without a framework.
func (h *Handlers) Register(mux routes.Registrar) {
	mux.HandleFunc("POST /v1/products", h.CreateProduct)
	mux.HandleFunc("GET /v1/products", h.ListProducts)
	mux.HandleFunc("GET /v1/products/{id}", h.GetProduct)
	mux.HandleFunc("PATCH /v1/products/{id}", h.UpdateProduct)
	mux.HandleFunc("POST /v1/products/{id}/images", h.AddProductImage)
	mux.HandleFunc("GET /v1/products/{id}/images", h.ListProductImages)
	mux.HandleFunc("DELETE /v1/products/{id}/images/{image_id}", h.DeleteProductImage)
	mux.HandleFunc("POST /v1/products/{id}/reviews", h.AddReview)
	mux.HandleFunc("GET /v1/products/{id}/reviews", h.GetReviews)
}
