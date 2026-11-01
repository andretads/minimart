package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	mux := newTestMux(t)

	// First create an account
	do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"loginuser","password":"password123","email":"loginuser@example.com"}`)

	t.Run("valid credentials set cookies and return user", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"loginuser","password":"password123"}`)
		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Success bool `json:"success"`
			Data    struct {
				ID                   string `json:"id"`
				Username             string `json:"username"`
				Email                string `json:"email"`
				CreatedAt            string `json:"created_at"`
				AccessTokenExpiresAt string `json:"access_token_expires_at"`
				ExpiresIn            int64  `json:"expires_in"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.True(t, env.Success)
		assert.Equal(t, "loginuser", env.Data.Username)
		assert.Equal(t, "loginuser@example.com", env.Data.Email)
		assert.NotEmpty(t, env.Data.AccessTokenExpiresAt)
		assert.Positive(t, env.Data.ExpiresIn)
		assert.LessOrEqual(t, env.Data.ExpiresIn, int64((15 * time.Minute).Seconds()))

		// Verify cookies are set
		cookies := rec.Result().Cookies()
		var accessCookie, refreshCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "mm_at" {
				accessCookie = c
			}
			if c.Name == "mm_rt" {
				refreshCookie = c
			}
		}
		require.NotNil(t, accessCookie, "access token cookie should be set")
		require.NotNil(t, refreshCookie, "refresh token cookie should be set")
		assert.True(t, accessCookie.HttpOnly, "access cookie should be HttpOnly")
		assert.True(t, refreshCookie.HttpOnly, "refresh cookie should be HttpOnly")
		assert.Equal(t, "/", accessCookie.Path)
		assert.Equal(t, "/", refreshCookie.Path)
		assert.NotEmpty(t, accessCookie.Value)
		assert.NotEmpty(t, refreshCookie.Value)
	})

	t.Run("remember me extends the refresh cookie lifetime", func(t *testing.T) {
		normal := do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"loginuser","password":"password123"}`)
		remembered := do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"loginuser","password":"password123","remember_me":true}`)

		refreshMaxAge := func(rec *httptest.ResponseRecorder) int {
			for _, c := range rec.Result().Cookies() {
				if c.Name == "mm_rt" {
					return c.MaxAge
				}
			}
			return 0
		}

		assert.Positive(t, refreshMaxAge(normal))
		assert.Greater(t, refreshMaxAge(remembered), refreshMaxAge(normal))
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"loginuser","password":"wrong"}`)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("unknown username returns 401 (no enumeration)", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"nobody","password":"password123"}`)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("empty username returns 401", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"","password":"password123"}`)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("empty password returns 401", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"loginuser","password":""}`)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("missing username returns 401", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/login", `{"password":"password123"}`)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("missing password returns 401", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"loginuser"}`)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/login", `{invalid json}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
