package http

import (
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
	"strconv"
)

func (h *Handlers) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	page := 1
	if v := r.URL.Query().Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "invalid page"))
			return
		}
		page = p
	}
	pageSize := 20
	if v := r.URL.Query().Get("page_size"); v != "" {
		ps, err := strconv.Atoi(v)
		if err != nil || ps < 1 || ps > 100 {
			response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "invalid page_size"))
			return
		}
		pageSize = ps
	}
	ns, total, err := h.svc.ListNotifications(r.Context(), userID, page, pageSize)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	out := make([]notificationResponse, 0, len(ns))
	for _, n := range ns {
		out = append(out, toNotificationResponse(n))
	}
	response.WriteJSON(w, http.StatusOK, response.OK(map[string]any{
		"data":        out,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": (total + pageSize - 1) / pageSize,
	}))
}
