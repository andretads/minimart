package http

import (
	"minimart/internal/category/application"
	"minimart/pkg/etag"
	"minimart/pkg/response"
	"net/http"
)

// DeleteCategory deletes a category by ID.
//
//	@Summary		Delete a category
//	@Description	Deletes a category by ID. Requires an exact If-Match header with the current version. Returns 204 on success, 400 for missing or invalid If-Match, 404 if not found, 409 for stale version.
//	@Tags			category
//	@Produce		json
//	@Param			id			path		string	true	"Category ID"
//	@Param			If-Match	header		string	true	"ETag from GET; must match current version"
//	@Success		204			"No content"
//	@Failure		400			{object}	response.Response{error=response.ErrorBody}
//	@Failure		404			{object}	response.Response{error=response.ErrorBody}
//	@Failure		409			{object}	response.Response{error=response.ErrorBody}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/categories/{id} [delete]
func (h *Handlers) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ifMatch := r.Header.Get("If-Match")
	if ifMatch == "" {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "cabeçalho If-Match obrigatório"))
		return
	}
	version, wildcard, ok := etag.Parse(ifMatch)
	if !ok || wildcard {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "cabeçalho If-Match inválido"))
		return
	}
	err := h.svc.DeleteCategory(r.Context(), application.DeleteCategoryRequest{
		ID:      id,
		Version: version,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
