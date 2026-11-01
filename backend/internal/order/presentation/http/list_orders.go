package http

import (
	"minimart/internal/order/application"
	"minimart/internal/order/domain"
	"minimart/pkg/response"
	"net/http"
	"strconv"
)

// ListOrders returns a paginated list of one user's orders, newest first.
//
//	@Summary		List a user's orders
//	@Description	Returns the orders owned by user_id, newest first. Page is 1-based.
//	@Tags			order
//	@Produce		json
//	@Param			user_id		query		string	true	"User ID"
//	@Param			page		query		int		false	"Page number (1-based, default 1)"
//	@Param			page_size	query		int		false	"Items per page (default 20, max 100)"
//	@Success		200			{object}	response.Response{data=[]orderResponse,meta=response.Meta}
//	@Failure		400			{object}	response.Response{error=response.ErrorBody}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/orders [get]
func (h *Handlers) ListOrders(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.svc.ListOrders(r.Context(), application.ListOrdersRequest{
		UserID:   q.Get("user_id"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}

	out := make([]orderResponse, 0, len(result.Items))
	for _, o := range result.Items {
		out = append(out, toOrderResponse(o))
	}
	response.WriteJSON(w, http.StatusOK, response.Paginated(out, response.Meta{
		Page:       result.Page,
		PageSize:   result.PageSize,
		Total:      result.Total,
		TotalPages: result.TotalPages,
	}))
}
