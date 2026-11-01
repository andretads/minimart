package http

import "minimart/internal/order/application"

type Handlers struct {
	svc *application.Service
}

func NewHandlers(svc *application.Service) *Handlers {
	return &Handlers{svc: svc}
}
