package authz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRoute(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/products", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("GET /v1/products/{id}", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("GET /v1/showcase/products/{id}", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("GET /livez", func(http.ResponseWriter, *http.Request) {})
	mux.Handle("/swagger/", http.NotFoundHandler())

	tests := []struct {
		name            string
		router          Router
		method          string
		target          string
		wantPath        string
		wantPattern     string
		wantMatched     bool
		wantIsPublic    bool
		wantOptional    bool
		wantGovernedBy  bool
		wantPermissions string
	}{
		{
			name:            "qualified pattern with a wildcard",
			router:          mux,
			method:          http.MethodGet,
			target:          "/v1/products/p1",
			wantPath:        "/v1/products/{id}",
			wantPattern:     "GET /v1/products/{id}",
			wantMatched:     true,
			wantIsPublic:    false,
			wantOptional:    false,
			wantGovernedBy:  true,
			wantPermissions: "catalog:read",
		},
		{
			name:            "qualified pattern without a wildcard",
			router:          mux,
			method:          http.MethodGet,
			target:          "/v1/products",
			wantPath:        "/v1/products",
			wantPattern:     "GET /v1/products",
			wantMatched:     true,
			wantGovernedBy:  true,
			wantPermissions: "catalog:read",
		},
		{
			name:            "public route with a wildcard",
			router:          mux,
			method:          http.MethodGet,
			target:          "/v1/showcase/products/p1",
			wantPath:        "/v1/showcase/products/{id}",
			wantPattern:     "GET /v1/showcase/products/{id}",
			wantMatched:     true,
			wantIsPublic:    true,
			wantGovernedBy:  false,
			wantPermissions: "",
		},
		{
			name:            "route registered without a method qualifier",
			router:          mux,
			method:          http.MethodGet,
			target:          "/livez",
			wantPath:        "/livez",
			wantPattern:     "GET /livez",
			wantMatched:     true,
			wantIsPublic:    true,
			wantGovernedBy:  false,
			wantPermissions: "",
		},
		{
			name:            "unregistered route falls back to the request path",
			router:          mux,
			method:          http.MethodGet,
			target:          "/v1/unknown",
			wantPath:        "/v1/unknown",
			wantPattern:     "GET /v1/unknown",
			wantMatched:     false,
			wantIsPublic:    false,
			wantGovernedBy:  false,
			wantPermissions: "",
		},
		{
			name:            "nil router falls back to the request path",
			router:          nil,
			method:          http.MethodGet,
			target:          "/v1/products/p1",
			wantPath:        "/v1/products/p1",
			wantPattern:     "GET /v1/products/p1",
			wantMatched:     false,
			wantIsPublic:    false,
			wantGovernedBy:  false,
			wantPermissions: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequestWithContext(t.Context(), tt.method, tt.target, nil)
			route := ResolveRoute(r, tt.router)

			assert.Equal(t, tt.wantPath, route.Path)
			assert.Equal(t, tt.wantPattern, route.Pattern)
			assert.Equal(t, tt.wantIsPublic, IsPublic(route))
			assert.Equal(t, tt.wantOptional, IsOptionalAuth(route))

			perm, ok := PolicyMap.RequiredPermission(route)
			assert.Equal(t, tt.wantGovernedBy, ok)
			assert.Equal(t, tt.wantPermissions, perm)
		})
	}
}

func TestResolveRoute_PrefixedPattern(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.Handle("/swagger/", http.NotFoundHandler())

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/swagger/index.html", nil)
	route := ResolveRoute(r, mux)

	assert.Equal(t, "/swagger/", route.Path)
	assert.Equal(t, "GET /swagger/", route.Pattern)
	assert.True(t, IsPublic(route))
}

func TestPolicyMap_RequiredPermission(t *testing.T) {
	t.Parallel()

	t.Run("returns the permission of a governed route", func(t *testing.T) {
		t.Parallel()

		perm, ok := PolicyMap.RequiredPermission(Route{Method: http.MethodPost, Path: "/v1/orders", Pattern: "POST /v1/orders"})
		require.True(t, ok)
		assert.Equal(t, "orders:write", perm)
	})

	t.Run("public routes are not governed", func(t *testing.T) {
		t.Parallel()

		_, ok := PolicyMap.RequiredPermission(Route{Method: http.MethodGet, Path: "/v1/showcase/products", Pattern: "GET /v1/showcase/products"})
		assert.False(t, ok)
	})

	t.Run("optional auth routes are not governed", func(t *testing.T) {
		t.Parallel()

		_, ok := PolicyMap.RequiredPermission(Route{Method: http.MethodGet, Path: "/v1/auth/me", Pattern: "GET /v1/auth/me"})
		assert.False(t, ok)
	})

	t.Run("an unknown route is not governed", func(t *testing.T) {
		t.Parallel()

		_, ok := PolicyMap.RequiredPermission(Route{Method: http.MethodGet, Path: "/v1/nope", Pattern: "GET /v1/nope"})
		assert.False(t, ok)
	})
}
