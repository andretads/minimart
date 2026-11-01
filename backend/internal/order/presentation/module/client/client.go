package client

import (
	"context"
	"errors"
	"minimart/pkg/message"
	"time"
)

var (
	ErrOrderNotFound          = errors.New("order: pedido não encontrado")
	ErrIdempotencyConflict    = errors.New("order: conflito de idempotência")
	ErrOrderInProgress        = errors.New("order: pedido em andamento")
	ErrInsufficientStock      = errors.New("order: estoque insuficiente")
	ErrProductNotFound        = errors.New("order: produto não encontrado")
	ErrNoItems                = errors.New("order: sem itens")
	ErrInvalidQuantity        = errors.New("order: quantidade inválida")
	ErrIdempotencyKeyRequired = errors.New("order: chave de idempotência obrigatória")
)

type PlaceOrderRequest struct {
	UserID              string
	IdempotencyKey      string
	Items               []LineItem
	ShippingQuoteID     string
	ShippingMethodID    string
	ShippingMethodName  string
	DestinationSnapshot DestinationSnapshot
	ShippingCostCents   int64
}

type LineItem struct {
	ProductID string
	Quantity  int
}

// DestinationSnapshot is the shared cross-domain delivery-address value type.
type DestinationSnapshot = message.DestinationSnapshot

// Order is the dependency-free view of an order passed across domains. The
// shipping fields are the snapshot taken at placement: they are empty on an
// order placed without shipping.
type Order struct {
	ID                  string
	UserID              string
	CustomerID          string
	Status              string
	TotalCents          int64
	Currency            string
	CreatedAt           time.Time
	ShippingQuoteID     string
	ShippingMethodID    string
	ShippingMethodName  string
	ShippingCostCents   int64
	DestinationSnapshot DestinationSnapshot
}

type OrderClient interface {
	PlaceOrder(ctx context.Context, req PlaceOrderRequest) (Order, error)
	GetOrder(ctx context.Context, id string) (Order, error)
}
