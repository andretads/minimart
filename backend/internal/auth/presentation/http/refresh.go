package http

import (
	"minimart/pkg/auth"
	"minimart/pkg/metrics"
	"minimart/pkg/response"
	"net/http"
)

// Refresh uses a refresh token from cookie to obtain a new access token.
//
//	@Summary		Refresh access token
//	@Description	Validates the refresh token from cookie, renews both the access and refresh token cookies (sliding session), and returns the new access-token expiry. Returns 401 if the refresh token is invalid or expired.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	response.Response{data=refreshResponse}
//	@Failure		401	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/auth/refresh [post]
func (h *Handlers) Refresh(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)

	// Read refresh token from cookie
	cookie, err := r.Cookie(auth.RefreshTokenCookieName)
	if err != nil {
		h.svc.RecordTokenRefresh(metrics.StatusError)
		writeUnauthorized(w, "refresh token not provided")
		return
	}

	// Decrypt refresh token
	refreshBytes, err := h.cookieEncryption.Decrypt(cookie.Value)
	if err != nil {
		h.svc.RecordTokenRefresh(metrics.StatusError)
		writeUnauthorized(w, "invalid refresh token")
		return
	}
	refreshToken := string(refreshBytes)

	claims, err := h.tokens.ValidateRefreshToken(refreshToken)
	if err != nil {
		h.svc.RecordTokenRefresh(metrics.StatusError)
		writeUnauthorized(w, "invalid or expired token")
		return
	}

	acc, err := h.svc.GetAccount(r.Context(), claims.UserID)
	if err != nil {
		h.svc.RecordTokenRefresh(metrics.StatusError)
		writeUnauthorized(w, "invalid or expired token")
		return
	}

	accessToken, accessExpiresAt, err := h.tokens.GenerateAccessTokenWithExpiry(acc.ID, acc.Username)
	if err != nil {
		h.svc.RecordTokenRefresh(metrics.StatusError)
		response.WriteJSON(w, http.StatusInternalServerError, response.Fail(http.StatusInternalServerError, "erro interno"))
		return
	}

	rememberMe := claims.RememberMe && h.rememberMeRefreshTTL > 0
	refreshTTL := h.cookieConfig.RefreshTTL
	if rememberMe {
		refreshTTL = h.rememberMeRefreshTTL
	}

	// Sliding session: renew the refresh token as well as the access token.
	refreshToken, err = h.tokens.GenerateRefreshTokenForSession(acc.ID, acc.Username, refreshTTL, rememberMe)
	if err != nil {
		h.svc.RecordTokenRefresh(metrics.StatusError)
		response.WriteJSON(w, http.StatusInternalServerError, response.Fail(http.StatusInternalServerError, "erro interno"))
		return
	}

	accessEncrypted, err := h.cookieEncryption.Encrypt([]byte(accessToken))
	if err != nil {
		h.svc.RecordTokenRefresh(metrics.StatusError)
		response.WriteJSON(w, http.StatusInternalServerError, response.Fail(http.StatusInternalServerError, "erro interno"))
		return
	}
	refreshEncrypted, err := h.cookieEncryption.Encrypt([]byte(refreshToken))
	if err != nil {
		h.svc.RecordTokenRefresh(metrics.StatusError)
		response.WriteJSON(w, http.StatusInternalServerError, response.Fail(http.StatusInternalServerError, "erro interno"))
		return
	}

	h.cookieConfig.SetCookie(w, auth.AccessTokenCookieName, accessEncrypted, h.cookieConfig.AccessTTL)
	h.cookieConfig.SetCookie(w, auth.RefreshTokenCookieName, refreshEncrypted, refreshTTL)

	h.svc.RecordTokenRefresh(metrics.StatusSuccess)
	response.WriteJSON(w, http.StatusOK, response.OK(toRefreshResponse(accessExpiresAt)))
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, msg))
}
