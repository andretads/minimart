package http

import (
	"minimart/internal/showcase/application"
	"minimart/internal/showcase/domain"
	"minimart/pkg/response"
	"net/http"
	"strconv"
)

// ListProductReviews returns the approved reviews of a product, paginated.
//
//	@Summary		List product reviews
//	@Description	Returns the approved reviews of a product, newest first, paginated.
//	@Tags			showcase
//	@Produce		json
//	@Param			id			path		string	true	"Product ID"
//	@Param			page		query		int		false	"Page number (1-based, default 1)"
//	@Param			page_size	query		int		false	"Items per page (default 20, max 100)"
//	@Success		200			{object}	response.Response{data=[]reviewResponse,meta=response.Meta}
//	@Failure		400			{object}	response.Response{error=response.ErrorBody}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/showcase/products/{id}/reviews [get]
func (h *Handlers) ListProductReviews(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	q := r.URL.Query()

	page := 1
	if v := q.Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			respondDomainError(w, domain.ErrInvalidPage)
			return
		}
		page = p
	}

	pageSize := 20
	if v := q.Get("page_size"); v != "" {
		ps, err := strconv.Atoi(v)
		if err != nil || ps < 1 || ps > 100 {
			respondDomainError(w, domain.ErrInvalidPageSize)
			return
		}
		pageSize = ps
	}

	result, err := h.svc.ListReviews(r.Context(), application.ListReviewsRequest{
		ProductID: id,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}

	out := make([]reviewResponse, 0, len(result.Items))
	for _, review := range result.Items {
		out = append(out, toReviewResponse(review))
	}

	response.WriteJSON(w, http.StatusOK, response.Paginated(out, response.Meta{
		Page:       result.Page,
		PageSize:   result.PageSize,
		Total:      result.Total,
		TotalPages: result.TotalPages,
	}))
}
