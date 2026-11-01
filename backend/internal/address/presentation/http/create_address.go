package http

import (
	"encoding/json"
	"minimart/internal/address/application"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

type createAddressRequest struct {
	Label     string `json:"label"`
	FullName  string `json:"full_name"`
	Street    string `json:"street"`
	City      string `json:"city"`
	State     string `json:"state"`
	Zip       string `json:"zip"`
	Country   string `json:"country"`
	Phone     string `json:"phone,omitempty"`
	IsDefault bool   `json:"is_default,omitempty"`
}

func (h *Handlers) CreateAddress(w http.ResponseWriter, r *http.Request) {
	var req createAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo JSON inválido"))
		return
	}
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
	a, err := h.svc.CreateAddress(r.Context(), application.CreateAddressRequest{
		CustomerID: cust.ID,
		Label:      req.Label,
		FullName:   req.FullName,
		Street:     req.Street,
		City:       req.City,
		State:      req.State,
		Zip:        req.Zip,
		Country:    req.Country,
		Phone:      req.Phone,
		IsDefault:  req.IsDefault,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	writeAddressResponse(w, http.StatusCreated, a)
}
