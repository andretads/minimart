package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublicPathsAreAccessibleWithoutAuth(t *testing.T) {
	mux := newTestMux(t)

	publicPaths := []string{
		"/livez",
		"/readyz",
		"/metrics",
		"/v1/auth/signup",
		"/v1/auth/login",
		"/v1/auth/logout",
		"/swagger/doc.json",
	}

	for _, path := range publicPaths {
		t.Run(path, func(t *testing.T) {
			var req *http.Request
			if path == "/v1/auth/signup" || path == "/v1/auth/login" || path == "/v1/auth/logout" {
				req = httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, nil)
			} else {
				req = httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			// Should not be 401
			assert.NotEqual(t, http.StatusUnauthorized, rec.Code, "path %s should not require auth", path)
		})
	}
}

func TestRefreshEndpointRequiresRefreshCookie(t *testing.T) {
	mux := newTestMux(t)

	// Refresh endpoint is public (no access token required) but requires refresh cookie
	t.Run("without refresh cookie returns 401", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/auth/refresh", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestPrivatePathsRequireAuth(t *testing.T) {
	mux := newTestMux(t)

	privatePaths := []string{
		"/v1/auth/me",
		"/v1/customers",
		"/v1/orders",
		"/v1/payments",
		"/v1/notifications",
	}

	for _, path := range privatePaths {
		t.Run(path, func(t *testing.T) {
			var req *http.Request
			if path == "/v1/orders" || path == "/v1/payments" || path == "/v1/customers" {
				req = httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, nil)
			} else {
				req = httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusUnauthorized, rec.Code, "path %s should require auth", path)
		})
	}
}
