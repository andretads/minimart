package http

import (
	"minimart/pkg/etag"
	"minimart/pkg/response"
	"net/http"
)

// DeactivateCustomer deactivates a customer.
//
//	@Summary		Deactivate customer
//	@Description	Prevents a customer from performing new operations that require an active customer. Idempotent.
//	@Tags			customer
//	@Produce		json
//	@Param			id	path		string	true	"Customer ID"
//	@Header			200		{string}	ETag	"New customer version"
//	@Success		200	{object}	response.Response{data=customerResponse}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/customers/{id}/deactivate [post]
func (h *Handlers) DeactivateCustomer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := h.svc.DeactivateCustomer(r.Context(), id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	w.Header().Set("ETag", etag.Format(c.Version))
	response.WriteJSON(w, http.StatusOK, response.OK(toCustomerResponse(c)))
}
