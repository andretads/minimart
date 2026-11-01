package http

import (
	customerclient "minimart/internal/customer/presentation/module/client"
	"minimart/internal/shipping/application"
)

type Handlers struct {
	svc      *application.Service
	customer customerclient.CustomerClient
}

func NewHandlers(svc *application.Service, customer customerclient.CustomerClient) *Handlers {
	return &Handlers{svc: svc, customer: customer}
}
