// Package apiroutes holds the cross-module test that proves every route the
// server exposes is governed by an authorization policy.
//
// The policy map is fail-closed: a route with no entry is denied to every
// caller. A hand-maintained list of routes can silently drift from the mux, so
// this package registers the routes of every domain module on a recorder and
// checks the result against the policy map.
package routes

import (
	"minimart/pkg/auth"
	"minimart/pkg/authz"
	"minimart/pkg/routes"
	"net/http"
	"net/url"
	"strings"
	"testing"

	addresshttp "minimart/internal/address/presentation/http"
	authhttp "minimart/internal/auth/presentation/http"
	avatarhttp "minimart/internal/avatar/presentation/http"
	carthttp "minimart/internal/cart/presentation/http"
	cataloghttp "minimart/internal/catalog/presentation/http"
	categoryhttp "minimart/internal/category/presentation/http"
	customerhttp "minimart/internal/customer/presentation/http"
	notificationhttp "minimart/internal/notification/presentation/http"
	orderhttp "minimart/internal/order/presentation/http"
	paymenthttp "minimart/internal/payment/presentation/http"
	shippinghttp "minimart/internal/shipping/presentation/http"
	showcasehttp "minimart/internal/showcase/presentation/http"
	wishlisthttp "minimart/internal/wishlist/presentation/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recorder captures the route patterns a module publishes instead of serving
// them.
type recorder struct {
	patterns []string
}

var _ routes.Registrar = (*recorder)(nil)

func (r *recorder) HandleFunc(pattern string, _ func(http.ResponseWriter, *http.Request)) {
	r.patterns = append(r.patterns, pattern)
}

// registeredPatterns returns the qualified route patterns of every domain
// module. The handlers are built without their services: only the patterns
// matter here.
func registeredPatterns(t *testing.T) []string {
	t.Helper()

	rec := &recorder{}
	addresshttp.NewHandlers(nil, nil).Register(rec)
	authhttp.NewHandlers(nil, nil, nil, auth.CookieConfig{}, 0).Register(rec)
	avatarhttp.NewHandlers(nil).Register(rec)
	carthttp.NewHandlers(nil).Register(rec)
	cataloghttp.NewHandlers(nil, nil).Register(rec)
	categoryhttp.NewHandlers(nil).Register(rec)
	customerhttp.NewHandlers(nil).Register(rec)
	notificationhttp.NewHandlers(nil).Register(rec)
	orderhttp.NewHandlers(nil).Register(rec)
	paymenthttp.NewHandlers(nil).Register(rec)
	shippinghttp.NewHandlers(nil, nil).Register(rec)
	showcasehttp.NewHandlers(nil).Register(rec)
	wishlisthttp.NewHandlers(nil).Register(rec)

	return rec.patterns
}

func TestEveryRegisteredRouteIsGoverned(t *testing.T) {
	t.Parallel()

	patterns := registeredPatterns(t)
	require.NotEmpty(t, patterns)

	for _, pattern := range patterns {
		t.Run(pattern, func(t *testing.T) {
			t.Parallel()

			method, path, qualified := strings.Cut(pattern, " ")
			require.True(t, qualified, "route pattern %q has no method qualifier", pattern)

			route := authz.Route{Method: method, Path: path, Pattern: pattern}
			if authz.IsPublic(route) || authz.IsOptionalAuth(route) {
				return
			}

			perm, governed := authz.PolicyMap.RequiredPermission(route)
			require.True(t, governed, "route %q has no policy entry", pattern)
			assert.NotEmpty(t, perm, "route %q has an empty permission", pattern)
		})
	}
}

func TestEveryPolicyEntryMatchesARegisteredRoute(t *testing.T) {
	t.Parallel()

	registered := make(map[string]struct{})
	for _, pattern := range registeredPatterns(t) {
		registered[pattern] = struct{}{}
	}

	for pattern := range authz.PolicyMap {
		t.Run(pattern, func(t *testing.T) {
			t.Parallel()

			_, ok := registered[pattern]
			assert.True(t, ok, "policy entry %q does not match a registered route", pattern)
		})
	}
}

func TestPublicRoutesArePublicForAuthenticationToo(t *testing.T) {
	t.Parallel()

	isPublic := auth.PublicPaths(auth.PublicRoutePaths...)

	for path := range authz.PublicPaths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			if path == "/swagger/" {
				// The auth middleware always lets the swagger tree through.
				return
			}

			r := &http.Request{Method: http.MethodGet, URL: mustURL(t, path)}
			assert.True(t, isPublic(r), "path %q is public for authorization but requires authentication", path)
		})
	}
}

func mustURL(t *testing.T, path string) *url.URL {
	t.Helper()

	u, err := url.Parse(path)
	require.NoError(t, err)
	return u
}
