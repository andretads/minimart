package http

import (
	"encoding/json"
	"minimart/internal/customer/application"
	"minimart/pkg/etag"
	"minimart/pkg/response"
	"net/http"
)

// CreateCustomer creates a customer record.
//
//	@Summary		Create a customer
//	@Description	Creates a customer record and assigns a unique CustomerID linked to the provided UserID.
//	@Tags			customer
//	@Accept			json
//	@Produce		json
//	@Param			customer	body		createCustomerRequest	true	"Customer to create"
//	@Header			201		{string}	ETag	"New customer version"
//	@Success		201		{object}	response.Response{data=customerResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/customers [post]
func (h *Handlers) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var req createCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo JSON inválido"))
		return
	}
	c, err := h.svc.CreateCustomer(r.Context(), application.CreateCustomerRequest{
		UserID:      req.UserID,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	w.Header().Set("ETag", etag.Format(c.Version))
	response.WriteJSON(w, http.StatusCreated, response.Created(toCustomerResponse(c)))
}
