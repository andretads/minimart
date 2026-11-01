// Package client is the only part of the catalog domain other domains may import.
// It exposes the published contract interface and its plain data types - no
// entities, no application services, no infrastructure.
package client

import (
	"context"
	"errors"
)

// ErrProductNotFound is returned by the contract when no product has the given ID.
// It is a client-level sentinel so callers never import the catalog's domain package.
var ErrProductNotFound = errors.New("catalog: produto não encontrado")

// ErrInsufficientStock is returned by ReserveStock when a requested product does
// not have enough units available.
var ErrInsufficientStock = errors.New("catalog: estoque insuficiente")

// CatalogClient is the contract the catalog domain publishes to the rest of the system.
type CatalogClient interface {
	GetProduct(ctx context.Context, id string) (Product, error)
	// ReserveStock holds each item's quantity for orderID. It is all-or-nothing
	// and idempotent: a shortage reserves nothing (ErrInsufficientStock), and a
	// repeated call for the same order does not reserve twice.
	ReserveStock(ctx context.Context, orderID string, items []LineItem) error
	// ReleaseStock returns any outstanding stock held for orderID. It is
	// idempotent, so it is safe to call more than once.
	ReleaseStock(ctx context.Context, orderID string) error
}

// Product is the dependency-free view of a catalog product passed across domains.
type Product struct {
	ID         string
	Name       string
	PriceCents int64
	SKU        string
	Details    []ProductDetail
}

// ProductDetail is a key-value attribute of a product (e.g., Color: Red, Size: Large).
type ProductDetail struct {
	Name  string
	Value string
}

// LineItem is one product and the quantity an order wants to reserve.
type LineItem struct {
	ProductID string
	Quantity  int
}
