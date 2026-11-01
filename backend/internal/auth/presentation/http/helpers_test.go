package http

import (
	"minimart/internal/auth/application"
	"minimart/internal/auth/infrastructure/memory"
	"minimart/pkg/auth"
	"minimart/pkg/metrics"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestMux(t *testing.T) http.Handler {
	t.Helper()
	repo := memory.New()
	svc := application.NewService(repo, nil, metrics.Noop{})
	tokens, _ := auth.NewTokenService("test-secret", 15*time.Minute, 7*24*time.Hour)
	cookieEncryption, _ := auth.NewCookieEncryption([]byte("0123456789abcdef0123456789abcdef"))
	cookieConfig := auth.DefaultCookieConfig(15*time.Minute, 7*24*time.Hour)
	mux := http.NewServeMux()
	NewHandlers(svc, tokens, cookieEncryption, cookieConfig, 30*24*time.Hour).Register(mux)
	// Apply auth middleware so protected endpoints work in tests
	public := auth.PublicPaths("/livez", "/readyz", "/metrics", "/v1/auth/signup", "/v1/auth/login", "/v1/auth/refresh", "/v1/auth/logout", "/swagger/")
	optional := auth.PublicPaths("/v1/auth/me")
	return auth.Middleware(tokens, cookieEncryption, cookieConfig, public, optional, mux, mux)
}

func findCookie(t *testing.T, rec *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	require.FailNow(t, "cookie not found", name)
	return nil
}

func do(t *testing.T, mux http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return doWithBody(t, mux, method, path, body)
}

func doWithBody(t *testing.T, mux http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(t.Context(), method, path, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}
