package http

import (
	"encoding/json"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/pkg/response"
	"net/http"
)

// UpdateZone handles PATCH /v1/shipping/zones/{id}
//
//	@Summary		Update a shipping zone
//	@Description	Update an existing shipping zone.
//	@Tags			shipping
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Shipping zone ID"
//	@Param			zone	body		UpdateZoneRequest	true	"Shipping zone"
//	@Success		200		{object}	response.Response{data=ShippingZoneResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/zones/{id} [patch]
func (h *Handlers) UpdateZone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateZoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondDomainError(w, domain.ErrInvalidInput)
		return
	}
	zone, err := h.svc.UpdateZone(r.Context(), id, application.UpdateZoneRequest{
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
	response.WriteJSON(w, http.StatusOK, response.OK(toZoneResponse(zone)))
}
