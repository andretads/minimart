package http

import (
	"encoding/json"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/pkg/response"
	"net/http"
	"time"
)

// AddTrackingEvent handles POST /v1/shipping/shipments/{id}/events
//
//	@Summary		Add a tracking event
//	@Description	Add a tracking event to a shipment
//	@Tags			shipping
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Shipment ID"
//	@Param			event	body		AddTrackingEventRequest	true	"Tracking event to add"
//	@Success		201		{object}	response.Response{data=TrackingEventResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/shipments/{id}/events [post]
func (h *Handlers) AddTrackingEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req AddTrackingEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondDomainError(w, domain.ErrInvalidInput)
		return
	}

	occurredAt, err := parseTime(req.OccurredAt)
	if err != nil {
		respondDomainError(w, domain.ErrInvalidInput)
		return
	}

	event, err := h.svc.AddTrackingEvent(r.Context(), application.AddTrackingEventRequest{
		ShipmentID:  id,
		Status:      domain.ShipmentStatus(req.Status),
		Description: req.Description,
		Location:    req.Location,
		OccurredAt:  occurredAt,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, response.Created(toTrackingEventResponse(event)))
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
