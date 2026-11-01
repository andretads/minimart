package http

import (
	"minimart/internal/showcase/domain"
	"minimart/pkg/response"
	"net/http"
)

// ListCategories returns all categories for public display.
//
//	@Summary		List categories
//	@Description	Returns every category. No authentication required.
//	@Tags			showcase
//	@Produce		json
//	@Success		200	{object}	response.Response{data=[]categoryResponse}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/showcase/categories [get]
func (h *Handlers) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.svc.ListCategories(r.Context())
	if err != nil {
		respondDomainError(w, err)
		return
	}
	out := make([]categoryResponse, 0, len(categories))
	for _, c := range categories {
		out = append(out, toCategoryResponse(c))
	}
	response.WriteJSON(w, http.StatusOK, response.OK(out))
}

func toCategoryResponse(c domain.Category) categoryResponse {
	return categoryResponse{
		ID:   c.ID,
		Name: c.Name,
		Icon: c.Icon,
	}
}
