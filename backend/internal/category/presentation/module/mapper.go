package module

import (
	"minimart/internal/category/domain"
	"minimart/internal/category/presentation/module/client"
)

// toClient maps a domain Category to the published client type.
func toClient(c domain.Category) client.CategoryDTO {
	return client.CategoryDTO{
		ID:   c.ID,
		Name: c.Name,
		Icon: c.Icon,
	}
}
