package http

import (
	"minimart/pkg/response"
	"net/http"
)

// DeliverShipment handles POST /v1/shipping/shipments/{id}/deliver
//
//	@Summary		Deliver a shipment
//	@Description	Mark a shipment as delivered
//	@Tags			shipping
//	@Produce		json
//	@Param			id	path		string	true	"Shipment ID"
//	@Success		200	{object}	response.Response{data=ShipmentResponse}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		409	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/shipments/{id}/deliver [post]
func (h *Handlers) DeliverShipment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	shipment, err := h.svc.DeliverShipment(r.Context(), id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toShipmentResponse(shipment)))
}
