// Package module adapts the catalog application service to the published
// client.Catalog contract. It is the only path other domains use to reach the
// catalog domain - they never touch its application, domain, or infrastructure.
package module

import (
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/presentation/module/client"
)

// Catalog implements client.Catalog by delegating to the application service and
// translating domain types into the published client types.
type Catalog struct {
	svc *application.Service
}

// Compile-time proof this adapter satisfies the published contract.
var _ client.CatalogClient = (*Catalog)(nil)

func New(svc *application.Service) *Catalog {
	return &Catalog{svc: svc}
}
