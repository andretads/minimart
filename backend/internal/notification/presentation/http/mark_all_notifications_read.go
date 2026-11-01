package http

import (
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

// MarkAllNotificationsRead marks every notification of the authenticated user as read.
//
//	@Summary		Mark all notifications as read
//	@Description	Marks all of the authenticated user's notifications as read.
//	@Tags			notification
//	@Produce		json
//	@Success		204
//	@Failure		401	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/notifications/read-all [post]
func (h *Handlers) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	if err := h.svc.MarkAllNotificationsRead(r.Context(), userID); err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusNoContent, response.OK(nil))
}
