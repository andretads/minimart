package http

import (
	"encoding/json"
	"minimart/internal/shipping/application"
	"minimart/internal/shipping/domain"
	"minimart/pkg/response"
	"net/http"
)

// CreateMethod handles POST /v1/shipping/methods
//
//	@Summary		Create a shipping method
//	@Description	Create a new shipping method.
//	@Tags			shipping
//	@Accept			json
//	@Produce		json
//	@Param			method	body		CreateMethodRequest	true	"Shipping method"
//	@Success		201		{object}	response.Response{data=ShippingMethodResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/methods [post]
func (h *Handlers) CreateMethod(w http.ResponseWriter, r *http.Request) {
	var req CreateMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondDomainError(w, domain.ErrInvalidInput)
		return
	}
	method, err := h.svc.CreateMethod(r.Context(), application.CreateMethodRequest{
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
	response.WriteJSON(w, http.StatusCreated, response.Created(toMethodResponse(method)))
}
