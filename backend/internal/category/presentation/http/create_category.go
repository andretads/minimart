package http

import (
	"encoding/json"
	"minimart/internal/category/application"
	"minimart/pkg/etag"
	"minimart/pkg/response"
	"net/http"
)

// CreateCategory creates a category.
//
//	@Summary		Create a category
//	@Description	Creates a category record with a unique ID.
//	@Tags			category
//	@Accept			json
//	@Produce		json
//	@Param			category	body		createCategoryRequest	true	"Category to create"
//	@Header			201		{string}	ETag	"New category version"
//	@Success		201		{object}	response.Response{data=categoryResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		409		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/categories [post]
func (h *Handlers) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req createCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo JSON inválido"))
		return
	}
	c, err := h.svc.CreateCategory(r.Context(), application.CreateCategoryRequest{
		Name: req.Name,
		Icon: req.Icon,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	w.Header().Set("ETag", etag.Format(c.Version))
	response.WriteJSON(w, http.StatusCreated, response.Created(toCategoryResponse(c)))
}
