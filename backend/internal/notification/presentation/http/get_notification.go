package http

import (
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

// GetNotification returns a single notification by ID for the authenticated user.
//
//	@Summary		Get a notification
//	@Description	Returns a notification by its ID, scoped to the authenticated user.
//	@Tags			notification
//	@Produce		json
//	@Param			id		path		string	true	"Notification ID"
//	@Success		200		{object}	response.Response{data=notificationResponse}
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/notifications/{id} [get]
func (h *Handlers) GetNotification(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	n, err := h.svc.GetNotification(r.Context(), id, userID)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(toNotificationResponse(n)))
}
