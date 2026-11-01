// Package module adapts the order application service to the published
// client.OrderClient contract. It is the only path other domains use to reach
// the order domain - they never touch its application, domain, or infrastructure.
package module

import (
	"minimart/internal/order/application"
	"minimart/internal/order/presentation/module/client"
)

// Order implements client.OrderClient by delegating to the application service and
// translating domain types into the published client types.
type Order struct {
	svc *application.Service
}

// Compile-time proof this adapter satisfies the published contract.
var _ client.OrderClient = (*Order)(nil)

func New(svc *application.Service) *Order {
	return &Order{svc: svc}
}
