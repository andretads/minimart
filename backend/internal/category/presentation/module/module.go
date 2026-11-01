// Package module adapts the category application service to the published
// client.Category contract. It is the only path other domains use to reach
// the category domain - they never touch its application, domain, or infrastructure.
package module

import (
	"minimart/internal/category/application"
	"minimart/internal/category/presentation/module/client"
)

// Category implements client.Category by delegating to the application service and
// translating domain types into the published client types.
type Category struct {
	svc *application.Service
}

// Compile-time proof this adapter satisfies the published contract.
var _ client.CategoryClient = (*Category)(nil)

func New(svc *application.Service) *Category {
	return &Category{svc: svc}
}
