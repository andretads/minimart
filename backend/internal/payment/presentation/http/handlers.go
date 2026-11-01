package http

import "minimart/internal/payment/application"

// Handlers serves the payment HTTP API.
type Handlers struct {
	svc *application.Service
}

// NewHandlers builds the payment HTTP handlers over the application service.
func NewHandlers(svc *application.Service) *Handlers {
	return &Handlers{svc: svc}
}
