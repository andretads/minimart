package http

import (
	"minimart/pkg/etag"
	"minimart/pkg/response"
	"net/http"
)

// GetCustomer returns a customer by ID.
//
//	@Summary		Get a customer
//	@Description	Returns a customer by its CustomerID. The response carries an ETag for the current version; send it back as If-None-Match to get a 304, or as If-Match on PATCH to guard the update.
//	@Tags			customer
//	@Produce		json
//	@Param			id				path		string	true	"Customer ID"
//	@Param			If-None-Match	header		string	false	"ETag from a previous GET; returns 304 when the customer is unchanged"
//	@Header			200				{string}	ETag	"Current customer version"
//	@Success		200				{object}	response.Response{data=customerResponse}
//	@Success		304				"Not modified - the customer still matches the supplied ETag"
//	@Failure		404				{object}	response.Response{error=response.ErrorBody}
//	@Failure		500				{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/customers/{id} [get]
func (h *Handlers) GetCustomer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := h.svc.GetCustomerByID(r.Context(), id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	w.Header().Set("ETag", etag.Format(c.Version))
	if etag.Matches(r.Header.Get("If-None-Match"), c.Version) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toCustomerResponse(c)))
}
