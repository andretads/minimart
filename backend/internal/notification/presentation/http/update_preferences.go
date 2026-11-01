package http

import (
	"encoding/json"
	"minimart/internal/notification/application"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

func (h *Handlers) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	var req updatePreferencesRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil || !req.complete() {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "invalid preferences"))
		return
	}
	p, err := h.svc.UpdatePreferences(r.Context(), application.UpdatePreferencesRequest{UserID: userID, EmailEnabled: *req.EmailEnabled, SMSEnabled: *req.SMSEnabled, PushEnabled: *req.PushEnabled, MarketingEnabled: *req.MarketingEnabled})
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toPreferencesResponse(p)))
}
