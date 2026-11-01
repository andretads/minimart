package http

import (
	"encoding/json"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/pkg/response"
	"net/http"
)

// CreateZone handles POST /v1/shipping/zones
//
//	@Summary		Create a shipping zone
//	@Description	Create a new shipping zone.
//	@Tags			shipping
//	@Accept			json
//	@Produce		json
//	@Param			zone	body		CreateZoneRequest	true	"Shipping zone"
//	@Success		201		{object}	response.Response{data=ShippingZoneResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/zones [post]
func (h *Handlers) CreateZone(w http.ResponseWriter, r *http.Request) {
	var req CreateZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondDomainError(w, domain.ErrInvalidInput)
		return
	}
	zone, err := h.svc.CreateZone(r.Context(), application.CreateZoneRequest{
		Name:            req.Name,
		Country:         req.Country,
		State:           req.State,
		City:            req.City,
		PostalCodeStart: req.PostalCodeStart,
		PostalCodeEnd:   req.PostalCodeEnd,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, response.Created(toZoneResponse(zone)))
}
