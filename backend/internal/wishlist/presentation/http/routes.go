package wishlisthttp

import "minimart/pkg/routes"

func (h *Handlers) Register(mux routes.Registrar) {
	mux.HandleFunc("GET /v1/wishlist", h.GetWishlist)
	mux.HandleFunc("POST /v1/wishlist/items", h.AddItem)
	mux.HandleFunc("DELETE /v1/wishlist/items/{product_id}", h.RemoveItem)
	mux.HandleFunc("DELETE /v1/wishlist", h.ClearWishlist)
}
