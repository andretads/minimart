package http

import "minimart/internal/customer/application"

// Handlers is the customer domain's HTTP entry point.
type Handlers struct {
	svc *application.Service
}

func NewHandlers(svc *application.Service) *Handlers {
	return &Handlers{svc: svc}
}
