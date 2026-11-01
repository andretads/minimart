package http

import (
	"encoding/json"
	"minimart/internal/category/application"
	"minimart/pkg/etag"
	"minimart/pkg/response"
	"net/http"
)

// UpdateCategory updates a category's name and icon by ID.
//
//	@Summary		Update a category
//	@Description	Updates a category's name and icon. The version is taken from the If-Match header when present, otherwise from the required body field - both must be the value last read from GET. Returns 400 when it is missing or non-positive, and 409 when another writer changed the category since that version was read.
//	@Tags			category
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string					true	"Category ID"
//	@Param			If-Match	header		string					false	"ETag from GET; overrides the body version"
//	@Param			category	body		updateCategoryRequest	true	"Fields to update"
//	@Header			200			{string}	ETag	"New category version"
//	@Success		200			{object}	response.Response{data=categoryResponse}
//	@Failure		400			{object}	response.Response{error=response.ErrorBody}
//	@Failure		404			{object}	response.Response{error=response.ErrorBody}
//	@Failure		409			{object}	response.Response{error=response.ErrorBody}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/categories/{id} [patch]
func (h *Handlers) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	var req updateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo JSON inválido"))
		return
	}
	// An If-Match precondition, when present, overrides the body version: it is
	// the standard way to feed back the ETag returned by GET. Only the exact
	// version tag is accepted - the lock needs a concrete version.
	if ifMatch := r.Header.Get("If-Match"); ifMatch != "" {
		version, wildcard, ok := etag.Parse(ifMatch)
		if !ok || wildcard {
			response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "cabeçalho If-Match inválido"))
			return
		}
		req.Version = version
	}
	c, err := h.svc.UpdateCategory(r.Context(), application.UpdateCategoryRequest{
		ID:      r.PathValue("id"),
		Name:    req.Name,
		Icon:    req.Icon,
		Version: req.Version,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	w.Header().Set("ETag", etag.Format(c.Version))
	response.WriteJSON(w, http.StatusOK, response.OK(toCategoryResponse(c)))
}
