package http

import (
	"minimart/internal/customer/domain"
	"time"
)

// createCustomerRequest is the body of POST /v1/customers.
type createCustomerRequest struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name,omitempty"`
} //	@name	CustomerCreateRequest

// updateCustomerRequest is the body of PATCH /v1/customers/{id}.
type updateCustomerRequest struct {
	DisplayName string `json:"display_name"`
	Version     int    `json:"version"`
} //	@name	CustomerUpdateRequest

// customerResponse is the JSON returned to clients.
type customerResponse struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	DisplayName   string  `json:"display_name"`
	Status        string  `json:"status"`
	Version       int     `json:"version"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	DeactivatedAt *string `json:"deactivated_at,omitempty"`
} //	@name	Customer

func toCustomerResponse(c domain.Customer) customerResponse {
	var deactivatedAt *string
	if c.DeactivatedAt != nil {
		s := c.DeactivatedAt.Format(time.RFC3339)
		deactivatedAt = &s
	}
	return customerResponse{
		ID:            c.ID,
		UserID:        c.UserID,
		DisplayName:   c.DisplayName,
		Status:        string(c.Status),
		Version:       c.Version,
		CreatedAt:     c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     c.UpdatedAt.Format(time.RFC3339),
		DeactivatedAt: deactivatedAt,
	}
}
