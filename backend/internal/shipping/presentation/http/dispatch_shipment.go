package http

import (
	"encoding/json"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/pkg/response"
	"net/http"
)

// DispatchShipment handles POST /v1/shipping/shipments/{id}/dispatch
//
//	@Summary		Dispatch a shipment
//	@Description	Mark a shipment as dispatched with tracking information
//	@Tags			shipping
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string					true	"Shipment ID"
//	@Param			dispatch	body		DispatchShipmentRequest	true	"Dispatch details"
//	@Success		200			{object}	response.Response{data=ShipmentResponse}
//	@Failure		400			{object}	response.Response{error=response.ErrorBody}
//	@Failure		404			{object}	response.Response{error=response.ErrorBody}
//	@Failure		409			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/shipments/{id}/dispatch [post]
func (h *Handlers) DispatchShipment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req DispatchShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondDomainError(w, domain.ErrInvalidInput)
		return
	}
	shipment, err := h.svc.DispatchShipment(r.Context(), id, application.DispatchShipmentRequest{
		TrackingCode: req.TrackingCode,
		Carrier:      req.Carrier,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toShipmentResponse(shipment)))
}
