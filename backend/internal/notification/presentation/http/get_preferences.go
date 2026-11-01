package http

import (
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

// GetPreferences returns the authenticated user's notification switches. A user
// who has never saved any receives the defaults.
//
//	@Summary		Get notification preferences
//	@Description	Returns the authenticated user's notification switches, or the defaults when none were saved.
//	@Tags			notification
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response{data=preferencesResponse}
//	@Failure		401	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/notifications/preferences [get]
func (h *Handlers) GetPreferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	p, err := h.svc.GetPreferences(r.Context(), userID)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toPreferencesResponse(p)))
}
