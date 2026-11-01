package http

import (
	"minimart/internal/address/application"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

func (h *Handlers) ListAddresses(w http.ResponseWriter, r *http.Request) {
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
	addrs, err := h.svc.ListAddresses(r.Context(), application.ListAddressesRequest{
		CustomerID: cust.ID,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	writeAddressListResponse(w, http.StatusOK, addrs)
}
