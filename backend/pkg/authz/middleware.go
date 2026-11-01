package authz

import (
	"context"
	"minimart/pkg/metrics"
	"minimart/pkg/response"
	"net/http"
	"slices"

	authpkg "minimart/pkg/auth"
)

type PermissionStore interface {
	PermissionsForAccount(ctx context.Context, accountID string) ([]string, error)
}

// Middleware authorizes a request against PolicyMap.
//
// The router is used to resolve the request to the route pattern that handles
// it, so that policy entries are written per route ("GET /v1/products/{id}")
// rather than per concrete URL path. The router is normally the mux the
// middleware ultimately wraps.
//
// The middleware is fail-closed: a route with no policy entry is denied, and
// only public routes skip the permission lookup entirely.
func Middleware(store PermissionStore, m metrics.Metrics, router Router, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := ResolveRoute(r, router)
		if route.Pattern != "" && r.Pattern == "" {
			r.Pattern = route.Pattern
		}

		if IsPublic(route) {
			next.ServeHTTP(w, r)
			return
		}

		if IsOptionalAuth(route) {
			next.ServeHTTP(w, r)
			return
		}

		claims, ok := authpkg.ClaimsFromContext(r.Context())
		if !ok || claims == nil {
			authpkg.WriteUnauthorized(w)
			return
		}

		requiredPerm, ok := PolicyMap.RequiredPermission(route)
		if !ok {
			m.IncAuthzDenied(metrics.StatusError)
			response.WriteJSON(w, http.StatusForbidden, response.Fail(http.StatusForbidden, "nenhuma permissão definida para a rota"))
			return
		}

		perms, err := store.PermissionsForAccount(r.Context(), claims.UserID)
		if err != nil {
			m.IncAuthzDenied(metrics.StatusError)
			response.WriteJSON(w, http.StatusInternalServerError, response.Fail(http.StatusInternalServerError, "falha na consulta de permissões"))
			return
		}

		if !slices.Contains(perms, requiredPerm) {
			m.IncAuthzDenied(metrics.StatusError)
			response.WriteJSON(w, http.StatusForbidden, response.Fail(http.StatusForbidden, "permissões insuficientes"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
