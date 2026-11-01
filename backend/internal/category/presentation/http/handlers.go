package http

import "minimart/internal/category/application"

// Handlers is the category domain's HTTP entry point.
type Handlers struct {
	svc *application.Service
}

func NewHandlers(svc *application.Service) *Handlers {
	return &Handlers{svc: svc}
}
