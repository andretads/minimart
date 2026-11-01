package http

import (
	"minimart/internal/showcase/application"
	"minimart/internal/showcase/domain"
	"minimart/pkg/response"
	"net/http"
	"strconv"
)

// ListProducts returns a paginated list of products, newest first, using keyset
// pagination.
//
//	@Summary		List products
//	@Description	Returns a keyset-paginated list of all products in the showcase, newest first. Can be filtered by category_id. Pass the next_cursor value from a previous response to fetch the next page.
//	@Tags			showcase
//	@Produce		json
//	@Param			page_size	query		int		false	"Items per page (default 20, max 100)"
//	@Param			cursor		query		string	false	"Opaque keyset cursor from a previous response"
//	@Param			category_id	query		string	false	"Filter by category ID"
//	@Success		200			{object}	response.Response{data=[]productResponse,meta=response.Meta}
//	@Failure		400			{object}	response.Response{error=response.ErrorBody}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/showcase/products [get]
func (h *Handlers) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	pageSize := 20
	if v := q.Get("page_size"); v != "" {
		ps, err := strconv.Atoi(v)
		if err != nil || ps < 1 || ps > 100 {
			respondDomainError(w, domain.ErrInvalidPageSize)
			return
		}
		pageSize = ps
	}

	cursor := q.Get("cursor")
	categoryID := q.Get("category_id")

	result, err := h.svc.ListProducts(r.Context(), application.ListProductsRequest{
		PageSize:   pageSize,
		Cursor:     cursor,
		CategoryID: categoryID,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}

	out := make([]productResponse, 0, len(result.Items))
	for _, p := range result.Items {
		primaryImageURL := ""
		if len(p.Images) > 0 {
			primaryImageURL = h.svc.PresignImageURL(r.Context(), p.Images[0].ObjectKey)
		}
		out = append(out, toProductSummaryResponse(p, primaryImageURL))
	}

	hasMore := result.HasMore
	response.WriteJSON(w, http.StatusOK, response.Paginated(out, response.Meta{
		PageSize:   result.PageSize,
		NextCursor: result.NextCursor,
		HasMore:    &hasMore,
	}))
}
