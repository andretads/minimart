package http

import (
	"encoding/json"
	"minimart/internal/auth/application"
	"minimart/pkg/auth"
	"minimart/pkg/response"
	"net/http"
)

// Login authenticates a user and sets access and refresh tokens as HttpOnly cookies.
//
//	@Summary		Authenticate user
//	@Description	Authenticates a user with username and password. Sets access and refresh tokens as HttpOnly cookies on success and returns the user with the access-token expiry.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			login	body		loginRequest	true	"Login credentials"
//	@Success		200		{object}	response.Response{data=sessionResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		401		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/auth/login [post]
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "corpo JSON inválido"))
		return
	}

	acc, err := h.svc.Authenticate(r.Context(), application.AuthenticateRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}

	accessToken, accessExpiresAt, err := h.tokens.GenerateAccessTokenWithExpiry(acc.ID, acc.Username)
	if err != nil {
		response.WriteJSON(w, http.StatusInternalServerError, response.Fail(http.StatusInternalServerError, "erro interno"))
		return
	}

	rememberMe := req.RememberMe && h.rememberMeRefreshTTL > 0
	refreshTTL := h.cookieConfig.RefreshTTL
	if rememberMe {
		refreshTTL = h.rememberMeRefreshTTL
	}

	refreshToken, err := h.tokens.GenerateRefreshTokenForSession(acc.ID, acc.Username, refreshTTL, rememberMe)
	if err != nil {
		response.WriteJSON(w, http.StatusInternalServerError, response.Fail(http.StatusInternalServerError, "erro interno"))
		return
	}

	// Encrypt and set cookies
	accessEncrypted, err := h.cookieEncryption.Encrypt([]byte(accessToken))
	if err != nil {
		response.WriteJSON(w, http.StatusInternalServerError, response.Fail(http.StatusInternalServerError, "erro interno"))
		return
	}
	refreshEncrypted, err := h.cookieEncryption.Encrypt([]byte(refreshToken))
	if err != nil {
		response.WriteJSON(w, http.StatusInternalServerError, response.Fail(http.StatusInternalServerError, "erro interno"))
		return
	}

	h.cookieConfig.SetCookie(w, auth.AccessTokenCookieName, accessEncrypted, h.cookieConfig.AccessTTL)
	h.cookieConfig.SetCookie(w, auth.RefreshTokenCookieName, refreshEncrypted, refreshTTL)

	// Tokens stay in HttpOnly cookies; the body carries the user plus the
	// access-token expiry so the SPA can schedule a proactive refresh.
	response.WriteJSON(w, http.StatusOK, response.OK(toSessionResponse(acc.ID, acc.Username, acc.Email, acc.CreatedAt, accessExpiresAt)))
}
