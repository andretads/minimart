package http

import "minimart/internal/notification/application"

// Handlers is the notification domain's HTTP entry point. It holds the
// application service - never a repository or a database type.
type Handlers struct {
	svc *application.Service
}

func NewHandlers(svc *application.Service) *Handlers { return &Handlers{svc: svc} }
