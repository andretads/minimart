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

func TestMe(t *testing.T) {
	mux := newTestMux(t)

	// Create account and login to get cookies
	do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"meuser","password":"password123","email":"meuser@example.com"}`)
	loginRec := do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"meuser","password":"password123"}`)

	// Extract cookies from login response
	cookies := loginRec.Result().Cookies()
	var accessCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "mm_at" {
			accessCookie = c
			break
		}
	}
	require.NotNil(t, accessCookie)

	t.Run("valid cookie returns account info with access_token_expires_at", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/auth/me", nil)
		req.AddCookie(accessCookie)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

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
		assert.Equal(t, "meuser", env.Data.Username)
		assert.Equal(t, "meuser@example.com", env.Data.Email)
		assert.NotEmpty(t, env.Data.ID)
		assert.NotEmpty(t, env.Data.AccessTokenExpiresAt, "access_token_expires_at should be present")
		assert.Positive(t, env.Data.ExpiresIn)

		// Verify the expires_at is a valid RFC3339 timestamp in the future
		expiresAt, err := time.Parse(time.RFC3339, env.Data.AccessTokenExpiresAt)
		require.NoError(t, err)
		assert.True(t, expiresAt.After(time.Now()), "expiry should be in the future")
	})

	t.Run("missing cookie returns 401", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/auth/me", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid access cookie returns 401", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/auth/me", nil)
		req.AddCookie(&http.Cookie{
			Name:     "mm_at",
			Value:    "invalid-encrypted-value",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("malformed access cookie returns 401", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/auth/me", nil)
		req.AddCookie(&http.Cookie{
			Name:     "mm_at",
			Value:    "not-base64!",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("expired access token returns 401", func(t *testing.T) {
		// Create a test mux with very short access TTL
		repo := memory.New()
		svc := application.NewService(repo, nil, metrics.Noop{})
		tokens, _ := auth.NewTokenService("test-secret", 1*time.Nanosecond, 7*24*time.Hour) // expired immediately
		cookieEncryption, _ := auth.NewCookieEncryption([]byte("0123456789abcdef0123456789abcdef"))
		cookieConfig := auth.DefaultCookieConfig(1*time.Nanosecond, 7*24*time.Hour)
		handler := http.NewServeMux()
		NewHandlers(svc, tokens, cookieEncryption, cookieConfig, 30*24*time.Hour).Register(handler)
		public := auth.PublicPaths("/livez", "/readyz", "/metrics", "/v1/auth/signup", "/v1/auth/login", "/v1/auth/refresh", "/swagger/")
		optional := auth.PublicPaths("/v1/auth/me")
		expiredMux := auth.Middleware(tokens, cookieEncryption, cookieConfig, public, optional, handler, handler)

		// Create account and login
		do(t, expiredMux, http.MethodPost, "/v1/auth/signup", `{"username":"expuser","password":"password123","email":"expuser@example.com"}`)
		loginRec := do(t, expiredMux, http.MethodPost, "/v1/auth/login", `{"username":"expuser","password":"password123"}`)

		cookies := loginRec.Result().Cookies()
		var expiredAccessCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "mm_at" {
				expiredAccessCookie = c
				break
			}
		}
		require.NotNil(t, expiredAccessCookie)

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/auth/me", nil)
		req.AddCookie(expiredAccessCookie)
		rec := httptest.NewRecorder()
		expiredMux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
