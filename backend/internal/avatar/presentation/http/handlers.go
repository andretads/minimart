package http

import (
	"minimart/internal/avatar/application"
)

// Handlers holds the avatar HTTP handlers.
type Handlers struct {
	svc *application.Service
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(svc *application.Service) *Handlers {
	return &Handlers{svc: svc}
}
