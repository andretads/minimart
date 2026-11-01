package http

import (
	"encoding/json"
	"minimart/internal/auth/application"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

// ChangePassword verifies the authenticated user's current password and stores
// a hash of the new one.
//
//	@Summary		Change password
//	@Description	Verifies the current password and replaces it with the new one. The new password must match its confirmation. Requires a valid access token cookie.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			changePassword	body		changePasswordRequest	true	"Password change payload"
//	@Success		200				{object}	response.Response{data=map[string]string}
//	@Failure		400				{object}	response.Response{error=response.ErrorBody}
//	@Failure		401				{object}	response.Response{error=response.ErrorBody}
//	@Failure		404				{object}	response.Response{error=response.ErrorBody}
//	@Failure		500				{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/auth/change-password [post]
func (h *Handlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo JSON inválido"))
		return
	}

	err := h.svc.ChangePassword(r.Context(), application.ChangePasswordRequest{
		AccountID:          userID,
		CurrentPassword:    req.CurrentPassword,
		NewPassword:        req.NewPassword,
		ConfirmNewPassword: req.ConfirmNewPassword,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, response.OK(map[string]string{"status": "password_changed"}))
}
