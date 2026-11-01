// Package module adapts the auth application service to the published
// client.AuthClient contract. It is the only path other domains use to reach
// the auth domain - they never touch its application, domain, or infrastructure.
package module

import (
	"minimart/internal/auth/application"
	"minimart/internal/auth/presentation/module/client"
)

// Auth implements client.AuthClient by delegating to the application service and
// translating domain types into the published client types.
type Auth struct {
	svc *application.Service
}

// Compile-time proof this adapter satisfies the published contract.
var _ client.AuthClient = (*Auth)(nil)

func New(svc *application.Service) *Auth {
	return &Auth{svc: svc}
}
