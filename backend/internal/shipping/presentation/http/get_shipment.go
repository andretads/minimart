package http

import (
	"minimart/internal/shipping/domain"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

// GetShipment handles GET /v1/shipping/shipments/{id}
//
//	@Summary		Get a shipment
//	@Description	Returns a shipment by its ID.
//	@Tags			shipping
//	@Produce		json
//	@Param			id	path		string	true	"Shipment ID"
//	@Success		200	{object}	response.Response{data=ShipmentResponse}
//	@Failure		401	{object}	response.Response{error=response.ErrorBody}
//	@Failure		403	{object}	response.Response{error=response.ErrorBody}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/shipments/{id} [get]
func (h *Handlers) GetShipment(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	id := r.PathValue("id")
	shipment, err := h.svc.GetShipment(r.Context(), id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if shipment.UserID != userID {
		respondDomainError(w, domain.ErrOwnershipMismatch)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toShipmentResponse(shipment)))
}
