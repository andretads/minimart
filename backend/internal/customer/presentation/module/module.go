// Package module adapts the customer application service to the published
// client.CustomerClient contract. It is the only path other domains use to reach
// the customer domain - they never touch its application, domain, or infrastructure.
package module

import (
	"minimart/internal/customer/application"
	"minimart/internal/customer/presentation/module/client"
)

// Customer implements client.CustomerClient by delegating to the application service and
// translating domain types into the published client types.
type Customer struct {
	svc *application.Service
}

// Compile-time proof this adapter satisfies the published contract.
var _ client.CustomerClient = (*Customer)(nil)

func New(svc *application.Service) *Customer {
	return &Customer{svc: svc}
}
