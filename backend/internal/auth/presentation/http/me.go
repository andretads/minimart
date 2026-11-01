package http

import (
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

// Me returns the current authenticated user's information.
//
//	@Summary		Get current user
//	@Description	Returns the authenticated user's account information and access-token expiry. Requires a valid access-token cookie.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response{data=sessionResponse}
//	@Failure		401	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/auth/me [get]
func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, "não autenticado"))
		return
	}

	acc, err := h.svc.GetAccount(r.Context(), userID)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	accessTokenExpiresAt, _ := auth.AccessTokenExpiresAtFromContext(r.Context())

	response.WriteJSON(w, http.StatusOK, response.OK(toSessionResponse(acc.ID, acc.Username, acc.Email, acc.CreatedAt, accessTokenExpiresAt)))
}
