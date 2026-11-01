package http

import (
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
	"strconv"
)

// AddReview adds a review for a product.
//
//	@Summary		Add a product review
//	@Description	Adds a new review for a product. The review is attributed to the authenticated customer; reviews start as pending for moderation.
//	@Tags			catalog
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		string	true	"Product ID"
//	@Param			review		body		reviewRequest	true	"Review data"
//	@Success		201			{object}	response.Response{data=reviewResponse}
//	@Failure		400			{object}	response.Response{error=response.ErrorBody}
//	@Failure		401			{object}	response.Response{error=response.ErrorBody}
//	@Failure		404			{object}	response.Response{error=response.ErrorBody}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/products/{id}/reviews [post]
func (h *Handlers) AddReview(w http.ResponseWriter, r *http.Request) {
	var req reviewRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeRequestError(w, err)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}

	cust, err := h.customer.EnsureCustomer(r.Context(), userID)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	review, err := h.svc.AddReview(r.Context(), application.AddReviewRequest{
		ProductID:   r.PathValue("id"),
		CustomerID:  cust.ID,
		Description: req.Description,
		Rating:      req.Rating,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, response.Created(toReviewResponse(review)))
}

// GetReviews returns approved reviews for a product, paginated.
//
//	@Summary		Get product reviews
//	@Description	Returns approved reviews for a product, newest first, paginated.
//	@Tags			catalog
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		string	true	"Product ID"
//	@Param			page		query		int		false	"Page number (1-based, default 1)"
//	@Param			page_size	query		int		false	"Items per page (default 20, max 100)"
//	@Success		200			{object}	response.Response{data=[]reviewResponse,meta=response.Meta}
//	@Failure		400			{object}	response.Response{error=response.ErrorBody}
//	@Failure		404			{object}	response.Response{error=response.ErrorBody}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/products/{id}/reviews [get]
func (h *Handlers) GetReviews(w http.ResponseWriter, r *http.Request) {
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
		ProductID: r.PathValue("id"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}

	out := make([]reviewResponse, 0, len(result.Items))
	for _, rev := range result.Items {
		out = append(out, toReviewResponse(rev))
	}

	response.WriteJSON(w, http.StatusOK, response.Paginated(out, response.Meta{
		Page:       result.Page,
		PageSize:   result.PageSize,
		Total:      result.Total,
		TotalPages: result.TotalPages,
	}))
}
