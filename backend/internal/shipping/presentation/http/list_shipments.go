package http

import (
	"errors"
	"minimart/internal/shipping/domain"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
	"strconv"
)

// ListShipments handles GET /v1/shipping/shipments
//
//	@Summary		List shipments
//	@Description	Returns shipments for a user or a single shipment by order_id.
//	@Tags			shipping
//	@Produce		json
//	@Param			order_id	query		string	false	"Filter by order ID"
//	@Param			page		query		int		false	"Page number (1-based, default 1)"
//	@Param			page_size	query		int		false	"Items per page (default 20, max 100)"
//	@Success		200			{object}	response.Response{data=[]ShipmentResponse,meta=response.Meta}
//	@Failure		400			{object}	response.Response{error=response.ErrorBody}
//	@Failure		401			{object}	response.Response{error=response.ErrorBody}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/shipping/shipments [get]
func (h *Handlers) ListShipments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page := 1
	if v := q.Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			respondDomainError(w, domain.ErrInvalidInput)
			return
		}
		page = p
	}

	pageSize := 20
	if v := q.Get("page_size"); v != "" {
		ps, err := strconv.Atoi(v)
		if err != nil || ps < 1 || ps > 100 {
			respondDomainError(w, domain.ErrInvalidInput)
			return
		}
		pageSize = ps
	}

	userID, _ := auth.UserIDFromContext(r.Context())
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}

	if orderID := q.Get("order_id"); orderID != "" {
		shipment, err := h.svc.GetShipmentByOrderID(r.Context(), orderID)
		if err == nil && shipment.UserID != userID {
			err = domain.ErrShipmentNotFound
		}
		if errors.Is(err, domain.ErrShipmentNotFound) {
			response.WriteJSON(w, http.StatusOK, response.Paginated([]ShipmentResponse{}, response.Meta{
				Page:       1,
				PageSize:   pageSize,
				Total:      0,
				TotalPages: 0,
			}))
			return
		}
		if err != nil {
			respondDomainError(w, err)
			return
		}
		response.WriteJSON(w, http.StatusOK, response.Paginated([]ShipmentResponse{toShipmentResponse(shipment)}, response.Meta{
			Page:       1,
			PageSize:   pageSize,
			Total:      1,
			TotalPages: 1,
		}))
		return
	}

	shipments, total, err := h.svc.ListShipments(r.Context(), userID, page, pageSize)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	out := make([]ShipmentResponse, len(shipments))
	for i, s := range shipments {
		out[i] = toShipmentResponse(s)
	}
	totalPages := (total + pageSize - 1) / pageSize
	response.WriteJSON(w, http.StatusOK, response.Paginated(out, response.Meta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}))
}
