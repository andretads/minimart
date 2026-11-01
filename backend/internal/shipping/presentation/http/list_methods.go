package http

import (
	"minimart/pkg/response"
	"net/http"
)

// ListMethods handles GET /v1/shipping/methods
//
//	@Summary		List shipping methods
//	@Description	Get all active shipping methods available for checkout
//	@Tags			shipping
//	@Produce		json
//	@Success		200	{object}	response.Response{data=[]ShippingMethodResponse}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/methods [get]
func (h *Handlers) ListMethods(w http.ResponseWriter, r *http.Request) {
	methods, err := h.svc.ListMethods(r.Context())
	if err != nil {
		respondDomainError(w, err)
		return
	}
	resp := make([]ShippingMethodResponse, len(methods))
	for i, m := range methods {
		resp[i] = toMethodResponse(m)
	}
	response.WriteJSON(w, http.StatusOK, response.OK(resp))
}
