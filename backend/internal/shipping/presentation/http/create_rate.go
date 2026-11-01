package http

import (
	"encoding/json"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/pkg/response"
	"net/http"
)

// CreateRate handles POST /v1/shipping/rates
//
//	@Summary		Create a shipping rate
//	@Description	Create a shipping rate for a zone and method pair.
//	@Tags			shipping
//	@Accept			json
//	@Produce		json
//	@Param			rate	body		CreateRateRequest	true	"Shipping rate"
//	@Success		201		{object}	response.Response{data=ShippingRateResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/rates [post]
func (h *Handlers) CreateRate(w http.ResponseWriter, r *http.Request) {
	var req CreateRateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondDomainError(w, domain.ErrInvalidInput)
		return
	}
	rate, err := h.svc.CreateRate(r.Context(), application.CreateRateRequest{
		ZoneID:             req.ZoneID,
		MethodID:           req.MethodID,
		BaseCostCents:      req.BaseCostCents,
		FreeThresholdCents: req.FreeThresholdCents,
		Active:             req.Active,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, response.Created(toRateResponse(rate)))
}
