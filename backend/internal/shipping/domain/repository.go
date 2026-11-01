package domain

import (
	"context"
)

type Repository interface {
	// Shipping Methods
	GetMethod(ctx context.Context, id string) (ShippingMethod, error)
	ListActiveMethods(ctx context.Context) ([]ShippingMethod, error)
	ListAllMethods(ctx context.Context) ([]ShippingMethod, error)
	CreateMethod(ctx context.Context, method ShippingMethod) error
	UpdateMethod(ctx context.Context, method ShippingMethod) error

	// Shipping Zones
	GetZone(ctx context.Context, id string) (ShippingZone, error)
	ListZones(ctx context.Context) ([]ShippingZone, error)
	CreateZone(ctx context.Context, zone ShippingZone) error
	UpdateZone(ctx context.Context, zone ShippingZone) error

	// Shipping Rates
	GetRate(ctx context.Context, id string) (ShippingRate, error)
	ListRates(ctx context.Context, zoneID, methodID string) ([]ShippingRate, error)
	ListRatesByZone(ctx context.Context, zoneID string) ([]ShippingRate, error)
	ListRatesByMethod(ctx context.Context, methodID string) ([]ShippingRate, error)
	CreateRate(ctx context.Context, rate ShippingRate) error
	UpdateRate(ctx context.Context, rate ShippingRate) error

	// Shipping Quotes
	GetQuote(ctx context.Context, id string) (ShippingQuote, error)
	// GetQuoteByIdempotencyKey returns the quote created under an idempotency
	// key, or ErrQuoteNotFound when none carries it. This is the authoritative
	// idempotency lookup: it holds even when the Redis cache is empty.
	GetQuoteByIdempotencyKey(ctx context.Context, key string) (ShippingQuote, error)
	CreateQuote(ctx context.Context, quote ShippingQuote) error
	UpdateQuote(ctx context.Context, quote ShippingQuote) error
	ExpireQuotes(ctx context.Context) error

	// Shipments
	GetShipment(ctx context.Context, id string) (Shipment, error)
	GetShipmentByOrderID(ctx context.Context, orderID string) (Shipment, error)
	ListShipments(ctx context.Context, userID string, page, pageSize int) ([]Shipment, int, error)
	CreateShipment(ctx context.Context, shipment Shipment) error
	// CreateShipmentForQuote persists a shipment created from quote, consumes the
	// quote, and records the shipment's domain events in the transactional
	// outbox. All three happen in one transaction, so a quote can never be spent
	// twice and a shipment never exists without its quote consumed.
	CreateShipmentForQuote(ctx context.Context, shipment Shipment, quote ShippingQuote) error
	UpdateShipment(ctx context.Context, shipment Shipment) error

	// Tracking Events
	AddTrackingEvent(ctx context.Context, event TrackingEvent) error
	GetTrackingEvents(ctx context.Context, shipmentID string) ([]TrackingEvent, error)
}
