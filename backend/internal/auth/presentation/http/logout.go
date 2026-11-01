package http

import (
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

// Logout clears authentication cookies.
//
//	@Summary		Logout user
//	@Description	Clears the authentication cookies, logging the user out.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	response.Response{data=map[string]string}
//	@Router			/v1/auth/logout [post]
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)
	h.cookieConfig.ClearCookie(w, auth.AccessTokenCookieName)
	h.cookieConfig.ClearCookie(w, auth.RefreshTokenCookieName)
	response.WriteJSON(w, http.StatusOK, response.OK(map[string]string{"status": "logged_out"}))
}
