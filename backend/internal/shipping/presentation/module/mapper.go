package module

import (
	"minimart/internal/shipping/domain"
	"time"

	shippingclient "minimart/internal/shipping/presentation/module/client"
)

// toClientShippingMethod maps a domain ShippingMethod to the client type.
func toClientShippingMethod(m domain.ShippingMethod) shippingclient.ShippingMethod {
	return shippingclient.ShippingMethod{
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

// toClientShippingZone maps a domain ShippingZone to the client type.
func toClientShippingZone(z domain.ShippingZone) shippingclient.ShippingZone {
	return shippingclient.ShippingZone{
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

// toClientShippingRate maps a domain ShippingRate to the client type.
func toClientShippingRate(r domain.ShippingRate) shippingclient.ShippingRate {
	return shippingclient.ShippingRate{
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

// toClientShippingQuote maps a domain ShippingQuote to the client type.
func toClientShippingQuote(q domain.ShippingQuote) shippingclient.ShippingQuote {
	return shippingclient.ShippingQuote{
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

// toClientShipment maps a domain Shipment to the client type.
func toClientShipment(sh domain.Shipment) shippingclient.Shipment {
	var shippedAt, deliveredAt *string
	if sh.ShippedAt != nil {
		t := sh.ShippedAt.Format(time.RFC3339)
		shippedAt = &t
	}
	if sh.DeliveredAt != nil {
		t := sh.DeliveredAt.Format(time.RFC3339)
		deliveredAt = &t
	}
	return shippingclient.Shipment{
		ID:                  sh.ID,
		OrderID:             sh.OrderID,
		UserID:              sh.UserID,
		MethodID:            sh.MethodID,
		TrackingCode:        sh.TrackingCode,
		Carrier:             sh.Carrier,
		ShippingCostCents:   sh.ShippingCostCents,
		DestinationSnapshot: sh.DestinationSnapshot,
		Status:              string(sh.Status),
		ShippedAt:           shippedAt,
		DeliveredAt:         deliveredAt,
		CreatedAt:           sh.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           sh.UpdatedAt.Format(time.RFC3339),
	}
}

// toClientTrackingEvent maps a domain TrackingEvent to the client type.
func toClientTrackingEvent(e domain.TrackingEvent) shippingclient.TrackingEvent {
	return shippingclient.TrackingEvent{
		ID:          e.ID,
		ShipmentID:  e.ShipmentID,
		Status:      string(e.Status),
		Description: e.Description,
		Location:    e.Location,
		OccurredAt:  e.OccurredAt.Format(time.RFC3339),
		CreatedAt:   e.CreatedAt.Format(time.RFC3339),
	}
}
