package module

import (
	"minimart/internal/customer/domain"
	"minimart/internal/customer/presentation/module/client"
)

// toClient maps a domain Customer to the published client type.
func toClient(c domain.Customer) client.Customer {
	return client.Customer{
		ID:          c.ID,
		UserID:      c.UserID,
		DisplayName: c.DisplayName,
		Status:      string(c.Status),
		CreatedAt:   c.CreatedAt,
	}
}
