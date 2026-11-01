package http

import (
	"minimart/internal/catalog/domain"
	"minimart/pkg/response"
	"net/http"
)

// ListProducts returns all products, optionally filtered by category.
//
//	@Summary		List products
//	@Description	Returns products in the catalog. Can be filtered by category_id query parameter. Each product includes its images ordered by position, with presigned URLs.
//	@Tags			catalog
//	@Produce		json
//	@Param			category_id	query		string	false	"Filter by category ID"
//	@Success		200			{object}	response.Response{data=[]productResponse}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/products [get]
func (h *Handlers) ListProducts(w http.ResponseWriter, r *http.Request) {
	categoryID := r.URL.Query().Get("category_id")
	var ps []domain.Product
	var err error
	if categoryID != "" {
		ps, err = h.svc.ListProductsByCategory(r.Context(), categoryID)
	} else {
		ps, err = h.svc.ListProducts(r.Context())
	}
	if err != nil {
		respondDomainError(w, err)
		return
	}
	productIDs := make([]string, 0, len(ps))
	for _, p := range ps {
		productIDs = append(productIDs, p.ID)
	}
	imagesByProduct, err := h.svc.ListProductImagesByProductIDs(r.Context(), productIDs)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	out := make([]productResponse, 0, len(ps))
	for _, p := range ps {
		out = append(out, toProductResponse(p, h.imageResponses(r.Context(), imagesByProduct[p.ID])))
	}
	response.WriteJSON(w, http.StatusOK, response.OK(out))
}
