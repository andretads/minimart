package authz

import (
	"context"
	"minimart/pkg/metrics"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authpkg "minimart/pkg/auth"
)

type stubPermissionStore struct {
	perms []string
	err   error
	calls int
}

func (s *stubPermissionStore) PermissionsForAccount(_ context.Context, _ string) ([]string, error) {
	s.calls++
	return s.perms, s.err
}

type countingMetrics struct {
	metrics.Noop
	denied int
}

func (c *countingMetrics) IncAuthzDenied(_ string) {
	c.denied++
}

func newTestMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /v1/products", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /v1/products/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /v1/showcase/products/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /v1/auth/me", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	return mux
}

func requestWithClaims(ctx context.Context, method, target string, withClaims bool) *http.Request {
	r := httptest.NewRequestWithContext(ctx, method, target, nil)
	if !withClaims {
		return r
	}
	claims := &authpkg.Claims{UserID: "account-1", Username: "tester"}
	return r.WithContext(authpkg.WithClaims(r.Context(), claims))
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		target     string
		perms      []string
		withClaims bool
		wantStatus int
		wantCalls  int
	}{
		{
			name:       "public route needs no token",
			method:     http.MethodGet,
			target:     "/livez",
			wantStatus: http.StatusOK,
			wantCalls:  0,
		},
		{
			name:       "public route with a wildcard needs no token",
			method:     http.MethodGet,
			target:     "/v1/showcase/products/p1",
			wantStatus: http.StatusOK,
			wantCalls:  0,
		},
		{
			name:       "protected route without a token is unauthorized",
			method:     http.MethodGet,
			target:     "/v1/products",
			wantStatus: http.StatusUnauthorized,
			wantCalls:  0,
		},
		{
			name:       "protected route with a token and the permission is allowed",
			method:     http.MethodGet,
			target:     "/v1/products",
			perms:      []string{"catalog:read"},
			withClaims: true,
			wantStatus: http.StatusOK,
			wantCalls:  1,
		},
		{
			name:       "protected route with a token and a different permission is forbidden",
			method:     http.MethodGet,
			target:     "/v1/products",
			perms:      []string{"orders:read"},
			withClaims: true,
			wantStatus: http.StatusForbidden,
			wantCalls:  1,
		},
		{
			name:       "protected route with a token and no permissions is forbidden",
			method:     http.MethodGet,
			target:     "/v1/products",
			perms:      nil,
			withClaims: true,
			wantStatus: http.StatusForbidden,
			wantCalls:  1,
		},
		{
			name:       "a wildcard route resolves to its policy entry",
			method:     http.MethodGet,
			target:     "/v1/products/p1",
			perms:      []string{"catalog:read"},
			withClaims: true,
			wantStatus: http.StatusOK,
			wantCalls:  1,
		},
		{
			name:       "optional auth route without a token is allowed",
			method:     http.MethodGet,
			target:     "/v1/auth/me",
			wantStatus: http.StatusOK,
			wantCalls:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := &stubPermissionStore{perms: tt.perms}
			mux := newTestMux()
			handler := Middleware(store, metrics.Noop{}, mux, mux)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, requestWithClaims(t.Context(), tt.method, tt.target, tt.withClaims))

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantCalls, store.calls)
		})
	}
}

func TestMiddleware_UngovernedRouteIsDenied(t *testing.T) {
	t.Parallel()

	store := &stubPermissionStore{perms: []string{"catalog:read"}}
	mux := newTestMux()
	handler := Middleware(store, metrics.Noop{}, mux, mux)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithClaims(t.Context(), http.MethodGet, "/v1/unknown", true))

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestMiddleware_PermissionLookupFailure(t *testing.T) {
	t.Parallel()

	store := &stubPermissionStore{err: assert.AnError}
	mux := newTestMux()
	handler := Middleware(store, metrics.Noop{}, mux, mux)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithClaims(t.Context(), http.MethodGet, "/v1/products", true))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestMiddleware_WithoutRouterFallsBackToThePath(t *testing.T) {
	t.Parallel()

	store := &stubPermissionStore{perms: []string{"catalog:read"}}
	handler := Middleware(store, metrics.Noop{}, nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithClaims(t.Context(), http.MethodGet, "/v1/products/p1", true))

	// Without a router the concrete path is used, which is not a policy key,
	// so the fail-closed default denies the request.
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestMiddleware_RecordsDeniedMetric(t *testing.T) {
	t.Parallel()

	recorder := &countingMetrics{}
	store := &stubPermissionStore{perms: []string{"orders:read"}}
	mux := newTestMux()
	handler := Middleware(store, recorder, mux, mux)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithClaims(t.Context(), http.MethodGet, "/v1/products", true))

	require.Equal(t, http.StatusForbidden, rec.Code)
	assert.Equal(t, 1, recorder.denied)
}
