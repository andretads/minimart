package http

import (
	"minimart/internal/address/application"
	customercontract "minimart/internal/customer/presentation/module/client"
)

type Handlers struct {
	svc      *application.Service
	customer customercontract.CustomerClient
}

func NewHandlers(svc *application.Service, customer customercontract.CustomerClient) *Handlers {
	return &Handlers{svc: svc, customer: customer}
}
