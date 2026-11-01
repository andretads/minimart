package http

import (
	"minimart/internal/address/application"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

func (h *Handlers) SetDefaultAddress(w http.ResponseWriter, r *http.Request) {
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
	a, err := h.svc.SetDefaultAddress(r.Context(), application.SetDefaultAddressRequest{
		ID:         id,
		CustomerID: cust.ID,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	writeAddressResponse(w, http.StatusOK, a)
}
