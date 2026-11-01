package http

import (
	"encoding/json"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/pkg/response"
	"net/http"
)

// UpdateRate handles PATCH /v1/shipping/rates/{id}
//
//	@Summary		Update a shipping rate
//	@Description	Update an existing shipping rate.
//	@Tags			shipping
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Shipping rate ID"
//	@Param			rate	body		UpdateRateRequest	true	"Shipping rate"
//	@Success		200		{object}	response.Response{data=ShippingRateResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/rates/{id} [patch]
func (h *Handlers) UpdateRate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateRateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondDomainError(w, domain.ErrInvalidInput)
		return
	}
	rate, err := h.svc.UpdateRate(r.Context(), id, application.UpdateRateRequest{
		BaseCostCents:      req.BaseCostCents,
		FreeThresholdCents: req.FreeThresholdCents,
		Active:             req.Active,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toRateResponse(rate)))
}
