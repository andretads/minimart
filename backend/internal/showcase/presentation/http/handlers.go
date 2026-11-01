package http

import "minimart/internal/showcase/application"

// Handlers holds the HTTP handlers for the showcase domain.
type Handlers struct {
	svc *application.Service
}

// NewHandlers constructs the showcase HTTP handlers.
func NewHandlers(svc *application.Service) *Handlers {
	return &Handlers{svc: svc}
}
