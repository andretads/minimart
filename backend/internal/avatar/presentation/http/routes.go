package http

import "minimart/pkg/routes"

func (h *Handlers) Register(mux routes.Registrar) {
	mux.HandleFunc("GET /v1/avatars/{account_id}", h.GetAvatar)
	mux.HandleFunc("PUT /v1/avatars/{account_id}", h.UploadAvatar)
}
