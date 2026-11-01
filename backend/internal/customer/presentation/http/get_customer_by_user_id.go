package http

import (
	"minimart/pkg/etag"
	"minimart/pkg/response"
	"net/http"
)

// GetCustomerByUserID resolves an authenticated user's UserID to their CustomerID.
//
//	@Summary		Get customer by user ID
//	@Description	Resolves the authenticated user's UserID to their CustomerID. The response carries an ETag for the current version; send it back as If-None-Match to get a 304.
//	@Tags			customer
//	@Produce		json
//	@Param			user_id	query		string	true	"User ID"
//	@Param			If-None-Match	header		string	false	"ETag from a previous GET; returns 304 when the customer is unchanged"
//	@Header			200				{string}	ETag	"Current customer version"
//	@Success		200				{object}	response.Response{data=customerResponse}
//	@Success		304				"Not modified - the customer still matches the supplied ETag"
//	@Failure		400				{object}	response.Response{error=response.ErrorBody}
//	@Failure		404				{object}	response.Response{error=response.ErrorBody}
//	@Failure		500				{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/customers [get]
func (h *Handlers) GetCustomerByUserID(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "parâmetro de consulta user_id é obrigatório"))
		return
	}
	c, err := h.svc.GetCustomerByUserID(r.Context(), userID)
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
