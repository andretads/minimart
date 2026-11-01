// Package http is the category's input adapter for HTTP. It translates between the
// wire (JSON) and the application layer, and never contains business rules.
package http

import (
	"minimart/internal/category/domain"
	"time"
)

// createCategoryRequest is the JSON body accepted by POST /v1/categories.
type createCategoryRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
} // @name CategoryCreateRequest

// updateCategoryRequest is the JSON body accepted by PATCH /v1/categories/{id}.
// Version is required and must be the value last read from GET - it is the
// optimistic lock the write is conditioned on.
type updateCategoryRequest struct {
	Name    string `json:"name"`
	Icon    string `json:"icon,omitempty"`
	Version int    `json:"version"`
} // @name CategoryUpdateRequest

// categoryResponse is the JSON returned to clients. It is kept separate from the
// domain entity so the wire contract stays stable even if the entity changes.
// Version is the optimistic lock a client reads back before its next update.
type categoryResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
} // @name Category

func toCategoryResponse(c domain.Category) categoryResponse {
	return categoryResponse{
		ID:        c.ID,
		Name:      c.Name,
		Icon:      c.Icon,
		Version:   c.Version,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
