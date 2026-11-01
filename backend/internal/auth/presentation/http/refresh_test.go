package http

import (
	"encoding/json"
	"minimart/internal/auth/application"
	"minimart/internal/auth/infrastructure/memory"
	"minimart/pkg/auth"
	"minimart/pkg/metrics"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefresh(t *testing.T) {
	mux := newTestMux(t)

	// Create account and login to get cookies
	do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"refreshuser","password":"password123","email":"refreshuser@example.com"}`)
	loginRec := do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"refreshuser","password":"password123"}`)

	// Extract cookies from login response
	cookies := loginRec.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "mm_rt" {
			refreshCookie = c
			break
		}
	}
	require.NotNil(t, refreshCookie)

	t.Run("valid refresh cookie returns new access token cookie", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/auth/refresh", nil)
		req.AddCookie(refreshCookie)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var env struct {
			Success bool `json:"success"`
			Data    struct {
				Status               string `json:"status"`
				AccessTokenExpiresAt string `json:"access_token_expires_at"`
				ExpiresIn            int64  `json:"expires_in"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		assert.True(t, env.Success)
		assert.Equal(t, "refreshed", env.Data.Status)
		assert.NotEmpty(t, env.Data.AccessTokenExpiresAt)
		assert.Positive(t, env.Data.ExpiresIn)
		assert.LessOrEqual(t, env.Data.ExpiresIn, int64((15 * time.Minute).Seconds()))

		// Verify new access and refresh token cookies are set
		cookies := rec.Result().Cookies()
		var accessCookie, renewedRefreshCookie *http.Cookie
		for _, c := range cookies {
			switch c.Name {
			case "mm_at":
				accessCookie = c
			case "mm_rt":
				renewedRefreshCookie = c
			}
		}
		require.NotNil(t, accessCookie)
		assert.NotEmpty(t, accessCookie.Value)
		assert.True(t, accessCookie.HttpOnly)
		assert.Equal(t, "/", accessCookie.Path)
		require.NotNil(t, renewedRefreshCookie, "refresh should renew the refresh cookie")
		assert.NotEmpty(t, renewedRefreshCookie.Value)
		assert.NotEqual(t, refreshCookie.Value, renewedRefreshCookie.Value)
	})

	t.Run("missing refresh cookie returns 401", func(t *testing.T) {
		rec := do(t, mux, http.MethodPost, "/v1/auth/refresh", "")
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid refresh cookie returns 401", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:     "mm_rt",
			Value:    "invalid-encrypted-value",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("malformed refresh cookie returns 401", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:     "mm_rt",
			Value:    "not-base64!",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("refresh renews the session and both cookie values still work", func(t *testing.T) {
		// First refresh using original refresh cookie
		req1 := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/auth/refresh", nil)
		req1.AddCookie(refreshCookie)
		rec1 := httptest.NewRecorder()
		mux.ServeHTTP(rec1, req1)
		require.Equal(t, http.StatusOK, rec1.Code)

		cookies1 := rec1.Result().Cookies()
		var newAccessCookie, renewedRefreshCookie *http.Cookie
		for _, c := range cookies1 {
			switch c.Name {
			case "mm_at":
				newAccessCookie = c
			case "mm_rt":
				renewedRefreshCookie = c
			}
		}
		require.NotNil(t, newAccessCookie)
		require.NotNil(t, renewedRefreshCookie)

		// The renewed refresh token works for the next refresh.
		req2 := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/auth/refresh", nil)
		req2.AddCookie(renewedRefreshCookie)
		rec2 := httptest.NewRecorder()
		mux.ServeHTTP(rec2, req2)
		require.Equal(t, http.StatusOK, rec2.Code)

		// Tokens are not revoked, so the original refresh token still works too.
		req3 := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/auth/refresh", nil)
		req3.AddCookie(refreshCookie)
		rec3 := httptest.NewRecorder()
		mux.ServeHTTP(rec3, req3)
		require.Equal(t, http.StatusOK, rec3.Code)
	})

	t.Run("expired refresh token returns 401", func(t *testing.T) {
		// Create a test mux with very short refresh TTL
		repo := memory.New()
		svc := application.NewService(repo, nil, metrics.Noop{})
		tokens, _ := auth.NewTokenService("test-secret", 15*time.Minute, 1*time.Nanosecond) // expired immediately
		cookieEncryption, _ := auth.NewCookieEncryption([]byte("0123456789abcdef0123456789abcdef"))
		cookieConfig := auth.DefaultCookieConfig(15*time.Minute, 1*time.Nanosecond)
		handler := http.NewServeMux()
		NewHandlers(svc, tokens, cookieEncryption, cookieConfig, 30*24*time.Hour).Register(handler)
		public := auth.PublicPaths("/livez", "/readyz", "/metrics", "/v1/auth/signup", "/v1/auth/login", "/v1/auth/refresh", "/swagger/")
		optional := auth.PublicPaths("/v1/auth/me")
		expiredMux := auth.Middleware(tokens, cookieEncryption, cookieConfig, public, optional, handler, handler)

		// Create account and login
		do(t, expiredMux, http.MethodPost, "/v1/auth/signup", `{"username":"expruser","password":"password123","email":"expruser@example.com"}`)
		loginRec := do(t, expiredMux, http.MethodPost, "/v1/auth/login", `{"username":"expruser","password":"password123"}`)

		cookies := loginRec.Result().Cookies()
		var expiredRefreshCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "mm_rt" {
				expiredRefreshCookie = c
				break
			}
		}
		require.NotNil(t, expiredRefreshCookie)

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/auth/refresh", nil)
		req.AddCookie(expiredRefreshCookie)
		rec := httptest.NewRecorder()
		expiredMux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
