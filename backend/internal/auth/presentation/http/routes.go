package http

import (
	"minimart/pkg/routes"
)

func (h *Handlers) Register(mux routes.Registrar) {
	mux.HandleFunc("POST /v1/auth/signup", h.SignUp)
	mux.HandleFunc("POST /v1/auth/login", h.Login)
	mux.HandleFunc("POST /v1/auth/refresh", h.Refresh)
	mux.HandleFunc("POST /v1/auth/logout", h.Logout)
	mux.HandleFunc("POST /v1/auth/change-password", h.ChangePassword)
	mux.HandleFunc("GET /v1/auth/me", h.Me)
}
