// Package module adapts the address application service to the published
// client.AddressClient contract. It is the only path other domains use to reach
// the address domain - they never touch its application, domain, or infrastructure.
package module

import (
	"minimart/internal/address/application"
	"minimart/internal/address/presentation/module/client"
)

// Address implements client.AddressClient by delegating to the application service and
// translating domain types into the published client types.
type Address struct {
	svc *application.Service
}

// Compile-time proof this adapter satisfies the published contract.
var _ client.AddressClient = (*Address)(nil)

func New(svc *application.Service) *Address {
	return &Address{svc: svc}
}
