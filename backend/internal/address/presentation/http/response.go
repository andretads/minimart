package http

import (
	"minimart/internal/address/domain"
	"minimart/pkg/etag"
	"minimart/pkg/response"
	"net/http"
	"time"
)

type AddressResponse struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	FullName  string `json:"full_name"`
	Street    string `json:"street"`
	City      string `json:"city"`
	State     string `json:"state"`
	Zip       string `json:"zip"`
	Country   string `json:"country"`
	Phone     string `json:"phone,omitempty"`
	IsDefault bool   `json:"is_default"`
	Version   int    `json:"version"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toAddressResponse(a domain.Address) AddressResponse {
	return AddressResponse{
		ID:        a.ID,
		Label:     a.Label,
		FullName:  a.FullName,
		Street:    a.Street,
		City:      a.City,
		State:     a.State,
		Zip:       a.Zip,
		Country:   a.Country,
		Phone:     a.Phone,
		IsDefault: a.IsDefault,
		Version:   a.Version,
		CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: a.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toAddressResponseList(addrs []domain.Address) []AddressResponse {
	resp := make([]AddressResponse, len(addrs))
	for i, a := range addrs {
		resp[i] = toAddressResponse(a)
	}
	return resp
}

func writeAddressResponse(w http.ResponseWriter, status int, a domain.Address) {
	w.Header().Set("ETag", etag.Format(a.Version))
	response.WriteJSON(w, status, response.OK(toAddressResponse(a)))
}

func writeAddressListResponse(w http.ResponseWriter, status int, addrs []domain.Address) {
	response.WriteJSON(w, status, response.OK(toAddressResponseList(addrs)))
}
