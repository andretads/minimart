package http

import (
	"encoding/json"
	"minimart/internal/address/application"
	"minimart/pkg/auth"
	"minimart/pkg/etag"
	"minimart/pkg/response"
	"net/http"
)

type updateAddressRequest struct {
	Label     string `json:"label"`
	FullName  string `json:"full_name"`
	Street    string `json:"street"`
	City      string `json:"city"`
	State     string `json:"state"`
	Zip       string `json:"zip"`
	Country   string `json:"country"`
	Phone     string `json:"phone,omitempty"`
	IsDefault bool   `json:"is_default,omitempty"`
	Version   int    `json:"version"`
}

func (h *Handlers) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req updateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo JSON inválido"))
		return
	}
	if ifMatch := r.Header.Get("If-Match"); ifMatch != "" {
		version, wildcard, ok := etag.Parse(ifMatch)
		if !ok || wildcard {
			response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "cabeçalho If-Match inválido"))
			return
		}
		req.Version = version
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
	a, err := h.svc.UpdateAddress(r.Context(), application.UpdateAddressRequest{
		ID:         id,
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
		Version:    req.Version,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	writeAddressResponse(w, http.StatusOK, a)
}
