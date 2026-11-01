// Package client is the only part of the category domain other domains may import.
// It exposes the published contract interface and its plain data types - no
// entities, no application services, no infrastructure.
package client

import (
	"context"
	"errors"
)

// ErrCategoryNotFound is returned by the contract when no category has the given ID.
// It is a client-level sentinel so callers never import the category's domain package.
var ErrCategoryNotFound = errors.New("category: categoria não encontrada")

// CategoryClient is the contract the category domain publishes to the rest of the system.
type CategoryClient interface {
	GetCategory(ctx context.Context, id string) (CategoryDTO, error)
}

// CategoryDTO is the dependency-free view of a category passed across domains.
type CategoryDTO struct {
	ID   string
	Name string
	Icon string
}
