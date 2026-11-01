package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	svc, _ := NewTokenService("test-secret", 15*time.Minute, 7*24*time.Hour)
	encryption, _ := NewCookieEncryption([]byte("0123456789abcdef0123456789abcdef"))
	cookieConfig := DefaultCookieConfig(15*time.Minute, 7*24*time.Hour)
	public := PublicPaths("/livez", "/readyz", "/metrics", "/v1/auth/signup", "/v1/auth/login", "/v1/auth/refresh")
	optional := PublicPaths("/v1/auth/me")

	mux := http.NewServeMux()
	// Public path handlers (return 200 OK)
	mux.HandleFunc("/livez", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/v1/auth/signup", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/v1/auth/login", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/v1/auth/refresh", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	// Swagger prefix handler
	mux.HandleFunc("/swagger/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	// Optional-auth path handler
	mux.HandleFunc("/v1/auth/me", func(w http.ResponseWriter, r *http.Request) {
		uid, ok := UserIDFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(uid))
	})
	// Private path handler
	mux.HandleFunc("/private", func(w http.ResponseWriter, r *http.Request) {
		uid, ok := UserIDFromContext(r.Context())
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(uid))
	})

	handler := Middleware(svc, encryption, cookieConfig, public, optional, mux, mux)

	// Helper to create a valid access token cookie
	createAccessCookie := func(t *testing.T, userID, username string) *http.Cookie {
		t.Helper()
		token := mustGenerateAccess(t, svc, userID, username)
		encrypted, err := encryption.Encrypt([]byte(token))
		require.NoError(t, err)
		return &http.Cookie{
			Name:     AccessTokenCookieName,
			Value:    encrypted,
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
	}

	t.Run("allows public paths without auth", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/livez", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("allows swagger prefix", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/swagger/doc.json", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("allows optional auth paths without a token", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/auth/me", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("injects claims on optional auth paths when a cookie is supplied", func(t *testing.T) {
		cookie := createAccessCookie(t, "user-1", "testuser")
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/auth/me", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "user-1", rec.Body.String())
	})

	t.Run("rejects missing cookie on private path", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/private", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "token de autenticação não fornecido")
	})

	t.Run("rejects invalid cookie", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/private", nil)
		req.AddCookie(&http.Cookie{
			Name:     AccessTokenCookieName,
			Value:    "invalid",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "token inválido")
	})

	t.Run("accepts valid cookie and injects claims", func(t *testing.T) {
		cookie := createAccessCookie(t, "user-1", "testuser")
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/private", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "user-1", rec.Body.String())
	})

	t.Run("sets WWW-Authenticate header on error", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/private", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, `Bearer error="invalid_token"`, rec.Header().Get("WWW-Authenticate"))
	})
}
