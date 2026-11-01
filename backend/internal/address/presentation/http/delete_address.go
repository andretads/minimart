package http

import (
	"minimart/internal/address/application"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

func (h *Handlers) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID, _ := auth.UserIDFromContext(r.Context())
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	cust, err := h.customer.EnsureCustomer(r.Context(), userID)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if err := h.svc.DeleteAddress(r.Context(), application.DeleteAddressRequest{
		ID:         id,
		CustomerID: cust.ID,
	}); err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusNoContent, response.OK(nil))
}
