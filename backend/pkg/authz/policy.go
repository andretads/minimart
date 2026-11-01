// Package authz maps routes to the permission required to call them.
//
// Two key spaces coexist and are deliberately distinct:
//
//   - PolicyMap is keyed by the qualified route pattern, e.g. "GET /v1/products".
//   - PublicPaths and OptionalAuthPaths are keyed by the path alone, e.g.
//     "/v1/products/{id}".
//
// Route resolution keeps the two apart: Route.Pattern carries the method
// qualifier, Route.Path does not.
package authz

import (
	"net/http"
	"strings"
)

// Policy maps a qualified route pattern to the permission required to call it.
type Policy map[string]string

// Router resolves a request to the route pattern that matched it.
// *http.ServeMux satisfies this interface through its Handler method.
type Router interface {
	Handler(r *http.Request) (h http.Handler, pattern string)
}

// Route is a request resolved to the route that handles it.
type Route struct {
	// Method is the HTTP method of the request.
	Method string
	// Path is the pattern path without its method qualifier, e.g.
	// "/v1/products/{id}". This is the key space of PublicPaths and
	// OptionalAuthPaths.
	Path string
	// Pattern is the qualified pattern, e.g. "GET /v1/products/{id}". This is
	// the key space of PolicyMap.
	Pattern string
}

// PublicPaths lists the routes that need neither a token nor a permission.
// Keys are pattern paths without a method qualifier.
var PublicPaths = map[string]struct{}{
	"/livez":                             {},
	"/readyz":                            {},
	"/metrics":                           {},
	"/swagger/":                          {},
	"/v1/auth/signup":                    {},
	"/v1/auth/login":                     {},
	"/v1/auth/refresh":                   {},
	"/v1/auth/logout":                    {},
	"/v1/showcase/products":              {},
	"/v1/showcase/categories":            {},
	"/v1/showcase/products/{id}":         {},
	"/v1/showcase/products/{id}/reviews": {},
}

// OptionalAuthPaths lists the routes that tolerate a missing token but still
// surface the claims when one is present. Keys are pattern paths without a
// method qualifier.
var OptionalAuthPaths = map[string]struct{}{
	"/v1/auth/me": {},
}

// PolicyMap maps a qualified route pattern to the permission required to call
// it. A route absent from this map is denied (fail-closed).
var PolicyMap = Policy{
	"GET /v1/products":                           "catalog:read",
	"GET /v1/products/{id}":                      "catalog:read",
	"POST /v1/products":                          "catalog:write",
	"PATCH /v1/products/{id}":                    "catalog:write",
	"POST /v1/products/{id}/images":              "catalog:write",
	"GET /v1/products/{id}/images":               "catalog:read",
	"DELETE /v1/products/{id}/images/{image_id}": "catalog:write",
	"GET /v1/categories":                         "catalog:read",
	"GET /v1/categories/{id}":                    "catalog:read",
	"POST /v1/categories":                        "catalog:write",
	"PATCH /v1/categories/{id}":                  "catalog:write",
	"DELETE /v1/categories/{id}":                 "catalog:write",
	"GET /v1/cart":                               "cart:read",
	"POST /v1/cart/items":                        "cart:write",
	"PATCH /v1/cart/items/{product_id}":          "cart:write",
	"DELETE /v1/cart/items/{product_id}":         "cart:write",
	"DELETE /v1/cart":                            "cart:write",
	"POST /v1/cart/checkout":                     "cart:write",
	"GET /v1/wishlist":                           "wishlist:read",
	"POST /v1/wishlist/items":                    "wishlist:write",
	"DELETE /v1/wishlist/items/{product_id}":     "wishlist:write",
	"DELETE /v1/wishlist":                        "wishlist:write",
	"POST /v1/orders":                            "orders:write",
	"GET /v1/orders":                             "orders:read",
	"POST /v1/orders/{id}/cancel":                "orders:write",
	"GET /v1/orders/{id}":                        "orders:read",
	"POST /v1/payments":                          "payments:write",
	"GET /v1/payments/{id}":                      "payments:read",
	"POST /v1/payments/{id}/refund":              "payments:write",
	"POST /v1/products/{id}/reviews":             "reviews:write",
	"GET /v1/products/{id}/reviews":              "catalog:read",
	"GET /v1/notifications":                      "notifications:read",
	"GET /v1/notifications/preferences":          "notifications:read",
	"PUT /v1/notifications/preferences":          "notifications:write",
	"GET /v1/notifications/{id}":                 "notifications:read",
	"POST /v1/notifications/{id}/read":           "notifications:write",
	"POST /v1/notifications/{id}/unread":         "notifications:write",
	"POST /v1/notifications/read-all":            "notifications:write",
	"DELETE /v1/notifications/{id}":              "notifications:write",
	"POST /v1/customers":                         "customers:write",
	"PATCH /v1/customers/{id}":                   "customers:write",
	"POST /v1/customers/{id}/deactivate":         "customers:write",
	"GET /v1/customers/{id}":                     "customers:read",
	"GET /v1/customers":                          "customers:read",
	"POST /v1/auth/change-password":              "accounts:write",
	"GET /v1/addresses":                          "addresses:read",
	"POST /v1/addresses":                         "addresses:write",
	"GET /v1/addresses/{id}":                     "addresses:read",
	"PATCH /v1/addresses/{id}":                   "addresses:write",
	"DELETE /v1/addresses/{id}":                  "addresses:write",
	"POST /v1/addresses/{id}/default":            "addresses:write",
	"GET /v1/avatars/{account_id}":               "avatars:read",
	"PUT /v1/avatars/{account_id}":               "avatars:write",
	"GET /v1/shipping/methods":                   "shipping:read",
	"POST /v1/shipping/quotes":                   "shipping:read",
	"GET /v1/shipping/quotes/{id}":               "shipping:read",
	"GET /v1/shipping/shipments":                 "shipping:read",
	"GET /v1/shipping/shipments/{id}":            "shipping:read",
	"GET /v1/shipping/shipments/{id}/events":     "shipping:read",
	"POST /v1/shipping/methods":                  "shipping:write",
	"PATCH /v1/shipping/methods/{id}":            "shipping:write",
	"POST /v1/shipping/zones":                    "shipping:write",
	"PATCH /v1/shipping/zones/{id}":              "shipping:write",
	"POST /v1/shipping/rates":                    "shipping:write",
	"PATCH /v1/shipping/rates/{id}":              "shipping:write",
	"POST /v1/shipping/shipments/{id}/process":   "shipping:write",
	"POST /v1/shipping/shipments/{id}/dispatch":  "shipping:write",
	"POST /v1/shipping/shipments/{id}/deliver":   "shipping:write",
	"POST /v1/shipping/shipments/{id}/events":    "shipping:write",
}

// RequiredPermission returns the permission required to call the route and
// whether the route is governed by the policy at all. Public and
// optional-auth routes are not governed and report false.
func (p Policy) RequiredPermission(route Route) (string, bool) {
	if _, ok := PublicPaths[route.Path]; ok {
		return "", false
	}
	if _, ok := OptionalAuthPaths[route.Path]; ok {
		return "", false
	}
	perm, ok := p[route.Pattern]
	return perm, ok
}

// IsPublic reports whether the route needs neither a token nor a permission.
func IsPublic(route Route) bool {
	_, ok := PublicPaths[route.Path]
	return ok
}

// IsOptionalAuth reports whether the route tolerates a missing token.
func IsOptionalAuth(route Route) bool {
	_, ok := OptionalAuthPaths[route.Path]
	return ok
}

// ResolveRoute resolves a request to the route that handles it.
//
// The router is the mux the request will eventually reach. It is consulted
// first so that the registered pattern is recovered with its wildcards
// ("GET /v1/products/{id}") instead of the concrete URL path
// ("/v1/products/p1"); without it, a policy entry could never be written for
// a parameterised route. When the router is nil or the request matches no
// route, the request's own path is used and the route is treated as
// ungoverned.
func ResolveRoute(r *http.Request, router Router) Route {
	pattern := ""
	if router != nil {
		_, pattern = router.Handler(r)
	}
	if pattern == "" {
		pattern = r.Pattern
	}
	if pattern == "" {
		return Route{Method: r.Method, Path: r.URL.Path, Pattern: r.Method + " " + r.URL.Path}
	}

	method, path, qualified := strings.Cut(pattern, " ")
	if !qualified || !isMethod(method) {
		// The registered pattern carries no method qualifier (e.g. "/swagger/"),
		// so the method of the request decides the qualified form.
		return Route{Method: r.Method, Path: pattern, Pattern: r.Method + " " + pattern}
	}
	return Route{Method: method, Path: path, Pattern: pattern}
}

// isMethod reports whether s looks like an HTTP method token.
func isMethod(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
