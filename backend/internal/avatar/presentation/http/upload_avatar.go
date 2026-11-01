package http

import (
	"minimart/internal/avatar/application"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

// UploadAvatar handles PUT /v1/avatars/{account_id}.
//
//	@Summary		Upload or replace avatar
//	@Description	Uploads an avatar image for the authenticated account. The account ID in the path must match the authenticated user.
//	@Tags			avatar
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			account_id	path		string	true	"Account ID"
//	@Param			image		formData	file	true	"Avatar image (max 5MB, image/jpeg, image/png, image/webp, image/gif)"
//	@Success		200			{object}	response.Response{data=avatarResponse}
//	@Failure		400			{object}	response.Response{error=response.ErrorBody}
//	@Failure		401			{object}	response.Response{error=response.ErrorBody}
//	@Failure		403			{object}	response.Response{error=response.ErrorBody}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/avatars/{account_id} [put]
func (h *Handlers) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("account_id")

	userID, _ := auth.UserIDFromContext(r.Context())
	if userID == "" {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}
	if userID != accountID {
		response.WriteJSON(w, http.StatusForbidden, response.Fail(http.StatusForbidden, "não é possível modificar o avatar de outra conta"))
		return
	}

	if !isMultipart(r) {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "Content-Type deve ser multipart/form-data"))
		return
	}

	if err := r.ParseMultipartForm(maxMultipartFormSize); err != nil { //nolint:gosec // bounded by maxMultipartFormSize
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "formulário multipart inválido"))
		return
	}

	img, closeFn, err := parseImageUpload(r)
	if closeFn != nil {
		defer closeFn()
	}
	if err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, err.Error()))
		return
	}

	view, err := h.svc.UploadAvatar(r.Context(), application.UploadAvatarRequest{
		AccountID: accountID,
		Image:     img,
	})
	if err != nil {
		respondDomainError(r.Context(), w, r, err)
		return
	}

	writeAvatar(w, http.StatusOK, view)
}
