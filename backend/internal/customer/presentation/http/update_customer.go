package http

import (
	"encoding/json"
	"minimart/internal/customer/application"
	"minimart/pkg/etag"
	"minimart/pkg/response"
	"net/http"
)

// UpdateCustomer updates a customer's profile.
//
//	@Summary		Update customer profile
//	@Description	Updates customer-owned profile fields such as the display name. The version is taken from the If-Match header when present, otherwise from the required body field - both must be the value last read from GET. Returns 400 when it is missing or non-positive, and 409 when another writer changed the customer since that version was read.
//	@Tags			customer
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string					true	"Customer ID"
//	@Param			If-Match	header		string					false	"ETag from GET; overrides the body version"
//	@Param			customer	body		updateCustomerRequest	true	"Profile fields to update"
//	@Header			200			{string}	ETag	"New customer version"
//	@Success		200			{object}	response.Response{data=customerResponse}
//	@Failure		400			{object}	response.Response{error=response.ErrorBody}
//	@Failure		404			{object}	response.Response{error=response.ErrorBody}
//	@Failure		409			{object}	response.Response{error=response.ErrorBody}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/customers/{id} [patch]
func (h *Handlers) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req updateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo JSON inválido"))
		return
	}
	// An If-Match precondition, when present, overrides the body version: it is
	// the standard way to feed back the ETag returned by GET. Only the exact
	// version tag is accepted - the lock needs a concrete version.
	if ifMatch := r.Header.Get("If-Match"); ifMatch != "" {
		version, wildcard, ok := etag.Parse(ifMatch)
		if !ok || wildcard {
			response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "cabeçalho If-Match inválido"))
			return
		}
		req.Version = version
	}
	c, err := h.svc.UpdateCustomerProfile(r.Context(), application.UpdateCustomerProfileRequest{
		ID:          id,
		DisplayName: req.DisplayName,
		Version:     req.Version,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	w.Header().Set("ETag", etag.Format(c.Version))
	response.WriteJSON(w, http.StatusOK, response.OK(toCustomerResponse(c)))
}
