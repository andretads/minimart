package http

import (
	"encoding/json"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/pkg/response"
	"net/http"
)

// UpdateMethod handles PATCH /v1/shipping/methods/{id}
//
//	@Summary		Update a shipping method
//	@Description	Update an existing shipping method.
//	@Tags			shipping
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Shipping method ID"
//	@Param			method	body		UpdateMethodRequest	true	"Shipping method"
//	@Success		200		{object}	response.Response{data=ShippingMethodResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/methods/{id} [patch]
func (h *Handlers) UpdateMethod(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req UpdateMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondDomainError(w, domain.ErrInvalidInput)
		return
	}
	method, err := h.svc.UpdateMethod(r.Context(), id, application.UpdateMethodRequest{
		Name:             req.Name,
		Description:      req.Description,
		Active:           req.Active,
		EstimatedMinDays: req.EstimatedMinDays,
		EstimatedMaxDays: req.EstimatedMaxDays,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toMethodResponse(method)))
}
