package http

import (
	"minimart/pkg/etag"
	"net/http"
)

// GetProduct returns a single product by ID.
//
//	@Summary		Get a product
//	@Description	Returns a product by its ID. The response carries an ETag for the current version; send it back as If-None-Match to get a 304, or as If-Match on PATCH to guard the update. Includes a presigned image URL if the product has an image.
//	@Tags			catalog
//	@Produce		json
//	@Param			id				path		string	true	"Product ID"
//	@Param			If-None-Match	header		string	false	"ETag from a previous GET; returns 304 when the product is unchanged"
//	@Header			200				{string}	ETag	"Current product version"
//	@Success		200				{object}	response.Response{data=productResponse}
//	@Success		304				"Not modified - the product still matches the supplied ETag"
//	@Failure		404				{object}	response.Response{error=response.ErrorBody}
//	@Failure		500				{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/products/{id} [get]
func (h *Handlers) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := h.svc.GetProduct(r.Context(), id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	w.Header().Set("ETag", etag.Format(p.Version))
	if etag.Matches(r.Header.Get("If-None-Match"), p.Version) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	h.writeProduct(w, r, http.StatusOK, p)
}
