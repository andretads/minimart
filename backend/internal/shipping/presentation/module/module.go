// Package module adapts the shipping application service to the published
// client.ShippingClient contract. It is the only path other domains use to reach
// the shipping domain - they never touch its application, domain, or infrastructure.
package module

import (
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/presentation/module/client"
)

// Shipping implements client.ShippingClient by delegating to the application service and
// translating domain types into the published client types.
type Shipping struct {
	svc *application.Service
}

// Compile-time proof this adapter satisfies the published contract.
var _ client.ShippingClient = (*Shipping)(nil)

func New(svc *application.Service) *Shipping {
	return &Shipping{svc: svc}
}
