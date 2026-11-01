package http

import (
	"minimart/pkg/etag"
	"minimart/pkg/response"
	"net/http"
)

// GetCategory returns a category by ID.
//
//	@Summary		Get a category
//	@Description	Returns a category by its ID. The response carries an ETag for the current version; send it back as If-None-Match to get a 304.
//	@Tags			category
//	@Produce		json
//	@Param			id				path		string	true	"Category ID"
//	@Param			If-None-Match	header		string	false	"ETag from a previous GET; returns 304 when the category is unchanged"
//	@Header			200				{string}	ETag	"Current category version"
//	@Success		200				{object}	response.Response{data=categoryResponse}
//	@Success		304				"Not modified - the category still matches the supplied ETag"
//	@Failure		404				{object}	response.Response{error=response.ErrorBody}
//	@Failure		500				{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/categories/{id} [get]
func (h *Handlers) GetCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := h.svc.GetCategory(r.Context(), id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	w.Header().Set("ETag", etag.Format(c.Version))
	if etag.Matches(r.Header.Get("If-None-Match"), c.Version) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toCategoryResponse(c)))
}
