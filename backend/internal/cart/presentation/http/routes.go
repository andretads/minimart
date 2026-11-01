package carthttp

import "minimart/pkg/routes"

func (h *Handlers) Register(mux routes.Registrar) {
	mux.HandleFunc("GET /v1/cart", h.GetCart)
	mux.HandleFunc("POST /v1/cart/items", h.AddItem)
	mux.HandleFunc("PATCH /v1/cart/items/{product_id}", h.UpdateItem)
	mux.HandleFunc("DELETE /v1/cart/items/{product_id}", h.RemoveItem)
	mux.HandleFunc("DELETE /v1/cart", h.ClearCart)
	mux.HandleFunc("POST /v1/cart/checkout", h.Checkout)
}
