package http

import (
	"minimart/pkg/response"
	"net/http"
)

// ProcessShipment handles POST /v1/shipping/shipments/{id}/process
//
//	@Summary		Process a shipment
//	@Description	Mark a shipment as being processed
//	@Tags			shipping
//	@Produce		json
//	@Param			id	path		string	true	"Shipment ID"
//	@Success		200	{object}	response.Response{data=ShipmentResponse}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		409	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/shipments/{id}/process [post]
func (h *Handlers) ProcessShipment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	shipment, err := h.svc.ProcessShipment(r.Context(), id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toShipmentResponse(shipment)))
}
