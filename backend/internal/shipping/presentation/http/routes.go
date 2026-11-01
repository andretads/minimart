package http

import "minimart/pkg/routes"

func (h *Handlers) Register(mux routes.Registrar) {
	mux.HandleFunc("GET /v1/shipping/methods", h.ListMethods)
	mux.HandleFunc("POST /v1/shipping/methods", h.CreateMethod)
	mux.HandleFunc("PATCH /v1/shipping/methods/{id}", h.UpdateMethod)

	mux.HandleFunc("POST /v1/shipping/zones", h.CreateZone)
	mux.HandleFunc("PATCH /v1/shipping/zones/{id}", h.UpdateZone)

	mux.HandleFunc("POST /v1/shipping/rates", h.CreateRate)
	mux.HandleFunc("PATCH /v1/shipping/rates/{id}", h.UpdateRate)

	mux.HandleFunc("POST /v1/shipping/quotes", h.CreateQuote)
	mux.HandleFunc("GET /v1/shipping/quotes/{id}", h.GetQuote)

	mux.HandleFunc("GET /v1/shipping/shipments", h.ListShipments)
	mux.HandleFunc("GET /v1/shipping/shipments/{id}", h.GetShipment)
	mux.HandleFunc("GET /v1/shipping/shipments/{id}/events", h.GetTrackingEvents)
	mux.HandleFunc("POST /v1/shipping/shipments/{id}/process", h.ProcessShipment)
	mux.HandleFunc("POST /v1/shipping/shipments/{id}/dispatch", h.DispatchShipment)
	mux.HandleFunc("POST /v1/shipping/shipments/{id}/deliver", h.DeliverShipment)
	mux.HandleFunc("POST /v1/shipping/shipments/{id}/events", h.AddTrackingEvent)
}
