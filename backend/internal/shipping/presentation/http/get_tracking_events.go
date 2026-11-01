package http

import (
	"minimart/internal/shipping/domain"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

// GetTrackingEvents handles GET /v1/shipping/shipments/{id}/events
//
//	@Summary		List shipment tracking events
//	@Description	Returns the tracking history of a shipment, oldest first. The shipment must belong to the authenticated user.
//	@Tags			shipping
//	@Produce		json
//	@Param			id	path		string	true	"Shipment ID"
//	@Success		200	{object}	response.Response{data=[]TrackingEventResponse}
//	@Failure		403	{object}	response.Response{error=response.ErrorBody}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/shipments/{id}/events [get]
func (h *Handlers) GetTrackingEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID, _ := auth.UserIDFromContext(r.Context())
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	shipment, err := h.svc.GetShipment(r.Context(), id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if shipment.UserID != userID {
		respondDomainError(w, domain.ErrOwnershipMismatch)
		return
	}
	events, err := h.svc.GetTrackingEvents(r.Context(), id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	out := make([]TrackingEventResponse, len(events))
	for i, e := range events {
		out[i] = toTrackingEventResponse(e)
	}
	response.WriteJSON(w, http.StatusOK, response.OK(out))
}
