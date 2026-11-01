package http

import (
	"minimart/internal/shipping/domain"
	"time"
)

// ShippingMethodResponse represents a shipping method response
// @name ShippingMethod
type ShippingMethodResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Active           bool   `json:"active"`
	EstimatedMinDays int    `json:"estimated_min_days"`
	EstimatedMaxDays int    `json:"estimated_max_days"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func toMethodResponse(m domain.ShippingMethod) ShippingMethodResponse {
	return ShippingMethodResponse{
		ID:               m.ID,
		Name:             m.Name,
		Description:      m.Description,
		Active:           m.Active,
		EstimatedMinDays: m.EstimatedMinDays,
		EstimatedMaxDays: m.EstimatedMaxDays,
		CreatedAt:        m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        m.UpdatedAt.Format(time.RFC3339),
	}
}

type ShippingZoneResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Country         string `json:"country"`
	State           string `json:"state"`
	City            string `json:"city"`
	PostalCodeStart string `json:"postal_code_start"`
	PostalCodeEnd   string `json:"postal_code_end"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

func toZoneResponse(z domain.ShippingZone) ShippingZoneResponse {
	return ShippingZoneResponse{
		ID:              z.ID,
		Name:            z.Name,
		Country:         z.Country,
		State:           z.State,
		City:            z.City,
		PostalCodeStart: z.PostalCodeStart,
		PostalCodeEnd:   z.PostalCodeEnd,
		CreatedAt:       z.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       z.UpdatedAt.Format(time.RFC3339),
	}
}

type ShippingRateResponse struct {
	ID                 string `json:"id"`
	ZoneID             string `json:"zone_id"`
	MethodID           string `json:"method_id"`
	BaseCostCents      int64  `json:"base_cost_cents"`
	FreeThresholdCents int64  `json:"free_threshold_cents"`
	Active             bool   `json:"active"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

func toRateResponse(r domain.ShippingRate) ShippingRateResponse {
	return ShippingRateResponse{
		ID:                 r.ID,
		ZoneID:             r.ZoneID,
		MethodID:           r.MethodID,
		BaseCostCents:      r.BaseCostCents,
		FreeThresholdCents: r.FreeThresholdCents,
		Active:             r.Active,
		CreatedAt:          r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          r.UpdatedAt.Format(time.RFC3339),
	}
}

type ShippingQuoteResponse struct {
	ID                  string                     `json:"id"`
	CustomerID          string                     `json:"customer_id"`
	MethodID            string                     `json:"method_id"`
	DestinationSnapshot domain.DestinationSnapshot `json:"destination_snapshot"`
	ShippingCostCents   int64                      `json:"shipping_cost_cents"`
	SubtotalCents       int64                      `json:"subtotal_cents"`
	FreeThresholdCents  int64                      `json:"free_threshold_cents"`
	EstimatedMinDays    int                        `json:"estimated_min_days"`
	EstimatedMaxDays    int                        `json:"estimated_max_days"`
	Status              string                     `json:"status"`
	ExpiresAt           string                     `json:"expires_at"`
	CreatedAt           string                     `json:"created_at"`
}

func toQuoteResponse(q domain.ShippingQuote) ShippingQuoteResponse {
	return ShippingQuoteResponse{
		ID:                  q.ID,
		CustomerID:          q.CustomerID,
		MethodID:            q.MethodID,
		DestinationSnapshot: q.DestinationSnapshot,
		ShippingCostCents:   q.ShippingCostCents,
		SubtotalCents:       q.SubtotalCents,
		FreeThresholdCents:  q.FreeThresholdCents,
		EstimatedMinDays:    q.EstimatedMinDays,
		EstimatedMaxDays:    q.EstimatedMaxDays,
		Status:              string(q.Status),
		ExpiresAt:           q.ExpiresAt.Format(time.RFC3339),
		CreatedAt:           q.CreatedAt.Format(time.RFC3339),
	}
}

type ShipmentResponse struct {
	ID                  string                     `json:"id"`
	OrderID             string                     `json:"order_id"`
	UserID              string                     `json:"user_id"`
	MethodID            string                     `json:"method_id"`
	TrackingCode        string                     `json:"tracking_code"`
	Carrier             string                     `json:"carrier"`
	ShippingCostCents   int64                      `json:"shipping_cost_cents"`
	DestinationSnapshot domain.DestinationSnapshot `json:"destination_snapshot"`
	Status              string                     `json:"status"`
	ShippedAt           *string                    `json:"shipped_at,omitempty"`
	DeliveredAt         *string                    `json:"delivered_at,omitempty"`
	CreatedAt           string                     `json:"created_at"`
	UpdatedAt           string                     `json:"updated_at"`
}

func toShipmentResponse(s domain.Shipment) ShipmentResponse {
	var shippedAt, deliveredAt *string
	if s.ShippedAt != nil {
		t := s.ShippedAt.Format(time.RFC3339)
		shippedAt = &t
	}
	if s.DeliveredAt != nil {
		t := s.DeliveredAt.Format(time.RFC3339)
		deliveredAt = &t
	}
	return ShipmentResponse{
		ID:                  s.ID,
		OrderID:             s.OrderID,
		UserID:              s.UserID,
		MethodID:            s.MethodID,
		TrackingCode:        s.TrackingCode,
		Carrier:             s.Carrier,
		ShippingCostCents:   s.ShippingCostCents,
		DestinationSnapshot: s.DestinationSnapshot,
		Status:              string(s.Status),
		ShippedAt:           shippedAt,
		DeliveredAt:         deliveredAt,
		CreatedAt:           s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           s.UpdatedAt.Format(time.RFC3339),
	}
}

type TrackingEventResponse struct {
	ID          string `json:"id"`
	ShipmentID  string `json:"shipment_id"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Location    string `json:"location"`
	OccurredAt  string `json:"occurred_at"`
	CreatedAt   string `json:"created_at"`
}

func toTrackingEventResponse(e domain.TrackingEvent) TrackingEventResponse {
	return TrackingEventResponse{
		ID:          e.ID,
		ShipmentID:  e.ShipmentID,
		Status:      string(e.Status),
		Description: e.Description,
		Location:    e.Location,
		OccurredAt:  e.OccurredAt.Format(time.RFC3339),
		CreatedAt:   e.CreatedAt.Format(time.RFC3339),
	}
}

type ListShipmentsResponse struct {
	Data []ShipmentResponse `json:"data"`
	Meta Meta               `json:"meta"`
}

type Meta struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Request DTOs
type CreateMethodRequest struct {
	Name             string `json:"name" validate:"required"`
	Description      string `json:"description"`
	Active           bool   `json:"active"`
	EstimatedMinDays int    `json:"estimated_min_days" validate:"required,min=0"`
	EstimatedMaxDays int    `json:"estimated_max_days" validate:"required,min=0"`
}

type UpdateMethodRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	Active           bool   `json:"active"`
	EstimatedMinDays int    `json:"estimated_min_days"`
	EstimatedMaxDays int    `json:"estimated_max_days"`
}

type CreateZoneRequest struct {
	Name            string `json:"name" validate:"required"`
	Country         string `json:"country" validate:"required"`
	State           string `json:"state"`
	City            string `json:"city"`
	PostalCodeStart string `json:"postal_code_start"`
	PostalCodeEnd   string `json:"postal_code_end"`
}

type UpdateZoneRequest struct {
	Name            string `json:"name"`
	Country         string `json:"country"`
	State           string `json:"state"`
	City            string `json:"city"`
	PostalCodeStart string `json:"postal_code_start"`
	PostalCodeEnd   string `json:"postal_code_end"`
}

type CreateRateRequest struct {
	ZoneID             string `json:"zone_id" validate:"required"`
	MethodID           string `json:"method_id" validate:"required"`
	BaseCostCents      int64  `json:"base_cost_cents" validate:"required,min=0"`
	FreeThresholdCents int64  `json:"free_threshold_cents" validate:"min=0"`
	Active             bool   `json:"active"`
}

type UpdateRateRequest struct {
	BaseCostCents      int64 `json:"base_cost_cents"`
	FreeThresholdCents int64 `json:"free_threshold_cents"`
	Active             bool  `json:"active"`
}

type CreateQuoteRequest struct {
	MethodID      string `json:"method_id" validate:"required"`
	AddressID     string `json:"address_id" validate:"required"`
	SubtotalCents int64  `json:"subtotal_cents" validate:"min=0"`
}

type DispatchShipmentRequest struct {
	TrackingCode string `json:"tracking_code" validate:"required"`
	Carrier      string `json:"carrier" validate:"required"`
}

type AddTrackingEventRequest struct {
	ShipmentID  string `json:"shipment_id" validate:"required"`
	Status      string `json:"status" validate:"required"`
	Description string `json:"description"`
	Location    string `json:"location"`
	OccurredAt  string `json:"occurred_at" validate:"required,rfc3339"`
}
