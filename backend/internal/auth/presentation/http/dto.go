package http

import (
	"time"
)

type signUpRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
} //	@name	AuthSignUpRequest

type loginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
} //	@name	AuthLoginRequest

type changePasswordRequest struct {
	CurrentPassword    string `json:"current_password"`
	NewPassword        string `json:"new_password"`
	ConfirmNewPassword string `json:"confirm_new_password"`
} //	@name	AuthChangePasswordRequest

type userInfo struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
} //	@name	AuthUser

type sessionResponse struct {
	ID                   string    `json:"id"`
	Username             string    `json:"username"`
	Email                string    `json:"email"`
	CreatedAt            time.Time `json:"created_at"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
	ExpiresIn            int64     `json:"expires_in"`
} //	@name	AuthSession

type refreshResponse struct {
	Status               string    `json:"status"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
	ExpiresIn            int64     `json:"expires_in"`
} //	@name	AuthRefreshResponse

func toUserInfo(id, username, email string, createdAt time.Time) *userInfo {
	return &userInfo{ID: id, Username: username, Email: email, CreatedAt: createdAt}
}

func toSessionResponse(id, username, email string, createdAt, accessTokenExpiresAt time.Time) sessionResponse {
	return sessionResponse{
		ID:                   id,
		Username:             username,
		Email:                email,
		CreatedAt:            createdAt,
		AccessTokenExpiresAt: accessTokenExpiresAt,
		ExpiresIn:            secondsUntil(accessTokenExpiresAt),
	}
}

func toRefreshResponse(accessTokenExpiresAt time.Time) refreshResponse {
	return refreshResponse{
		Status:               "refreshed",
		AccessTokenExpiresAt: accessTokenExpiresAt,
		ExpiresIn:            secondsUntil(accessTokenExpiresAt),
	}
}

func secondsUntil(t time.Time) int64 {
	seconds := int64(time.Until(t).Seconds())
	if seconds < 0 {
		return 0
	}
	return seconds
}
