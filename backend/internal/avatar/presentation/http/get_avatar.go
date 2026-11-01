package http

import (
	"minimart/internal/avatar/application"
	"net/http"
)

// GetAvatar handles GET /v1/avatars/{account_id}.
//
//	@Summary		Get avatar
//	@Description	Returns the avatar metadata and a presigned URL for the image.
//	@Tags			avatar
//	@Produce		json
//	@Param			account_id	path		string	true	"Account ID"
//	@Success		200			{object}	response.Response{data=avatarResponse}
//	@Failure		404			{object}	response.Response{error=response.ErrorBody}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/avatars/{account_id} [get]
func (h *Handlers) GetAvatar(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("account_id")

	view, err := h.svc.GetAvatar(r.Context(), application.GetAvatarRequest{
		AccountID: accountID,
	})
	if err != nil {
		respondDomainError(r.Context(), w, r, err)
		return
	}

	writeAvatar(w, http.StatusOK, view)
}
