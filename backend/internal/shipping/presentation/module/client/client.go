// Package client is the only part of the shipping domain other domains may import.
// It exposes the published contract interface and its plain data types - no
// entities, no application services, no infrastructure.
package client

import (
	"context"
	"errors"
	"minimart/pkg/message"
)

// ErrQuoteNotFound is returned when no quote has the given ID.
var ErrQuoteNotFound = errors.New("shipping: cotação de frete não encontrada")

// ErrShipmentNotFound is returned when no shipment has the given ID.
var ErrShipmentNotFound = errors.New("shipping: envio não encontrado")

// ErrQuoteExpired is returned when a quote has expired.
var ErrQuoteExpired = errors.New("shipping: cotação de frete expirada")

// ErrQuoteConsumed is returned when a quote has already been consumed.
var ErrQuoteConsumed = errors.New("shipping: cotação de frete consumida")

// ErrInvalidQuote is returned when a quote is invalid for the requested operation.
var ErrInvalidQuote = errors.New("shipping: cotação de frete inválida")

// ErrMethodNotFound is returned when no shipping method has the given ID.
var ErrMethodNotFound = errors.New("shipping: método de frete não encontrado")

// ErrZoneNotFound is returned when no shipping zone has the given ID.
var ErrZoneNotFound = errors.New("shipping: zona de frete não encontrada")

// ErrRateNotFound is returned when no shipping rate has the given ID.
var ErrRateNotFound = errors.New("shipping: tarifa de frete não encontrada")

// ErrNoMatchingRate is returned when no rate matches the given zone and method.
var ErrNoMatchingRate = errors.New("shipping: nenhuma tarifa de frete corresponde ao destino")

// ErrMethodInactive is returned when a shipping method is not active.
var ErrMethodInactive = errors.New("shipping: método de frete inativo")

// ErrZoneInactive is returned when a shipping zone is not active.
var ErrZoneInactive = errors.New("shipping: zona de frete inativa")

// ErrRateInactive is returned when a shipping rate is not active.
var ErrRateInactive = errors.New("shipping: tarifa de frete inativa")

// ErrIdempotencyConflict is returned when the idempotency key already belongs
// to a quote request with different parameters.
var ErrIdempotencyConflict = errors.New("shipping: conflito de idempotência")

// ShippingMethod is the dependency-free view of a shipping method passed across domains.
type ShippingMethod struct {
	ID               string
	Name             string
	Description      string
	Active           bool
	EstimatedMinDays int
	EstimatedMaxDays int
	CreatedAt        string
	UpdatedAt        string
}

// ShippingZone is the dependency-free view of a shipping zone passed across domains.
type ShippingZone struct {
	ID              string
	Name            string
	Country         string
	State           string
	City            string
	PostalCodeStart string
	PostalCodeEnd   string
	CreatedAt       string
	UpdatedAt       string
}

// ShippingRate is the dependency-free view of a shipping rate passed across domains.
type ShippingRate struct {
	ID                 string
	ZoneID             string
	MethodID           string
	BaseCostCents      int64
	FreeThresholdCents int64
	Active             bool
	CreatedAt          string
	UpdatedAt          string
}

// DestinationSnapshot is the shared cross-domain delivery-address value type.
type DestinationSnapshot = message.DestinationSnapshot

// ShippingQuote is the dependency-free view of a shipping quote passed across domains.
type ShippingQuote struct {
	ID                  string
	CustomerID          string
	MethodID            string
	DestinationSnapshot DestinationSnapshot
	ShippingCostCents   int64
	SubtotalCents       int64
	FreeThresholdCents  int64
	EstimatedMinDays    int
	EstimatedMaxDays    int
	Status              string
	ExpiresAt           string
	CreatedAt           string
}

// Shipment is the dependency-free view of a shipment passed across domains.
type Shipment struct {
	ID                  string
	OrderID             string
	UserID              string
	MethodID            string
	TrackingCode        string
	Carrier             string
	ShippingCostCents   int64
	DestinationSnapshot DestinationSnapshot
	Status              string
	ShippedAt           *string
	DeliveredAt         *string
	CreatedAt           string
	UpdatedAt           string
}

// TrackingEvent is the dependency-free view of a tracking event passed across domains.
type TrackingEvent struct {
	ID          string
	ShipmentID  string
	Status      string
	Description string
	Location    string
	OccurredAt  string
	CreatedAt   string
}

// GetQuoteRequest is the request to get a quote by ID.
type GetQuoteRequest struct {
	ID string
}

// ValidateQuoteRequest is the request to validate a quote.
type ValidateQuoteRequest struct {
	ID         string
	CustomerID string
}

// PriceQuoteRequest is the request to price a quote for a subtotal.
type PriceQuoteRequest struct {
	ID            string
	CustomerID    string
	SubtotalCents int64
}

// CreateQuoteRequest is the request to create a quote.
type CreateQuoteRequest struct {
	CustomerID    string
	MethodID      string
	AddressID     string
	SubtotalCents int64
	// IdempotencyKey makes the request safe to retry: a repeated call with the
	// same key and parameters returns the quote that already exists.
	IdempotencyKey string
}

// CreateMethodRequest is the request to create a shipping method.
type CreateMethodRequest struct {
	Name             string
	Description      string
	Active           bool
	EstimatedMinDays int
	EstimatedMaxDays int
}

// UpdateMethodRequest is the request to update a shipping method.
type UpdateMethodRequest struct {
	Name             string
	Description      string
	Active           bool
	EstimatedMinDays int
	EstimatedMaxDays int
}

// CreateZoneRequest is the request to create a shipping zone.
type CreateZoneRequest struct {
	Name            string
	Country         string
	State           string
	City            string
	PostalCodeStart string
	PostalCodeEnd   string
}

// UpdateZoneRequest is the request to update a shipping zone.
type UpdateZoneRequest struct {
	Name            string
	Country         string
	State           string
	City            string
	PostalCodeStart string
	PostalCodeEnd   string
}

// CreateRateRequest is the request to create a shipping rate.
type CreateRateRequest struct {
	ZoneID             string
	MethodID           string
	BaseCostCents      int64
	FreeThresholdCents int64
	Active             bool
}

// UpdateRateRequest is the request to update a shipping rate.
type UpdateRateRequest struct {
	BaseCostCents      int64
	FreeThresholdCents int64
	Active             bool
}

// ShippingClient is the contract the shipping domain publishes to the rest of the system.
type ShippingClient interface {
	ListMethods(ctx context.Context) ([]ShippingMethod, error)
	GetMethod(ctx context.Context, id string) (ShippingMethod, error)
	CreateMethod(ctx context.Context, req CreateMethodRequest) (ShippingMethod, error)
	UpdateMethod(ctx context.Context, id string, req UpdateMethodRequest) (ShippingMethod, error)

	ListZones(ctx context.Context) ([]ShippingZone, error)
	GetZone(ctx context.Context, id string) (ShippingZone, error)
	CreateZone(ctx context.Context, req CreateZoneRequest) (ShippingZone, error)
	UpdateZone(ctx context.Context, id string, req UpdateZoneRequest) (ShippingZone, error)

	ListRates(ctx context.Context, zoneID, methodID string) ([]ShippingRate, error)
	GetRate(ctx context.Context, id string) (ShippingRate, error)
	CreateRate(ctx context.Context, req CreateRateRequest) (ShippingRate, error)
	UpdateRate(ctx context.Context, id string, req UpdateRateRequest) (ShippingRate, error)

	GetQuote(ctx context.Context, req GetQuoteRequest) (ShippingQuote, error)
	CreateQuote(ctx context.Context, req CreateQuoteRequest) (ShippingQuote, error)
	ValidateQuote(ctx context.Context, req ValidateQuoteRequest) (ShippingQuote, error)
	PriceQuote(ctx context.Context, req PriceQuoteRequest) (ShippingQuote, error)

	GetShipment(ctx context.Context, id string) (Shipment, error)
	GetShipmentByOrderID(ctx context.Context, orderID string) (Shipment, error)
	ListShipments(ctx context.Context, userID string, page, pageSize int) ([]Shipment, int, error)

	GetTrackingEvents(ctx context.Context, shipmentID string) ([]TrackingEvent, error)
}
