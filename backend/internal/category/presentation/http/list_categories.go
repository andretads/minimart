package http

import (
	"minimart/pkg/response"
	"net/http"
)

// ListCategories returns all categories.
//
//	@Summary		List categories
//	@Description	Returns every category.
//	@Tags			category
//	@Produce		json
//	@Success		200	{object}	response.Response{data=[]categoryResponse}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/categories [get]
func (h *Handlers) ListCategories(w http.ResponseWriter, r *http.Request) {
	cs, err := h.svc.ListCategories(r.Context())
	if err != nil {
		respondDomainError(w, err)
		return
	}
	out := make([]categoryResponse, 0, len(cs))
	for _, c := range cs {
		out = append(out, toCategoryResponse(c))
	}
	response.WriteJSON(w, http.StatusOK, response.OK(out))
}
