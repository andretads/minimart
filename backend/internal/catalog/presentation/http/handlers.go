package http

import (
	"minimart/internal/catalog/application"
	customercontract "minimart/internal/customer/presentation/module/client"
)

// Handlers holds the catalog HTTP handlers and their dependencies: the service
// and the customer contract used to resolve the authenticated caller.
type Handlers struct {
	svc      *application.Service
	customer customercontract.CustomerClient
}

func NewHandlers(svc *application.Service, customer customercontract.CustomerClient) *Handlers {
	return &Handlers{svc: svc, customer: customer}
}
