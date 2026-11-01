package http

import (
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

// DeleteNotification removes a notification for the authenticated user.
//
//	@Summary		Delete a notification
//	@Description	Deletes the notification identified by id for the authenticated user.
//	@Tags			notification
//	@Produce		json
//	@Param			id	path	string	true	"Notification ID"
//	@Success		204
//	@Failure		401	{object}	response.Response{error=response.ErrorBody}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/notifications/{id} [delete]
func (h *Handlers) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	if err := h.svc.DeleteNotification(r.Context(), id, userID); err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusNoContent, response.OK(nil))
}
