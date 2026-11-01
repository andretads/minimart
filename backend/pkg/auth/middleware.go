package auth

import (
	"minimart/pkg/response"
	"net/http"
	"strings"
)

type PublicPredicate func(*http.Request) bool

// Router resolves a request to the route pattern that matched it.
// *http.ServeMux satisfies this interface through its Handler method.
type Router interface {
	Handler(r *http.Request) (h http.Handler, pattern string)
}

// Middleware validates the access token from cookie and puts the resulting
// claims in the request context.
//
// A request to a public path is passed through untouched. A request to an
// optional-auth path is passed through when it carries no token, and otherwise
// validated like any other request. Every other path requires a valid token.
func Middleware(tokens *TokenService, encryption *CookieEncryption, cookieCfg CookieConfig, isPublic, isOptional PublicPredicate, router Router, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublic(r) {
			next.ServeHTTP(w, r)
			return
		}

		if router != nil {
			_, pattern := router.Handler(r)
			if pattern != "" && r.Pattern == "" {
				r.Pattern = pattern
			}
		}

		// Read access token from cookie
		cookie, err := r.Cookie(AccessTokenCookieName)
		if err != nil {
			if isOptional(r) {
				next.ServeHTTP(w, r)
				return
			}
			writeUnauthorized(w, "token de autenticação não fornecido")
			return
		}

		// Decrypt cookie value
		tokenBytes, err := encryption.Decrypt(cookie.Value)
		if err != nil {
			writeUnauthorized(w, "token inválido")
			return
		}
		tokenString := string(tokenBytes)

		claims, err := tokens.ValidateAccessToken(tokenString)
		if err != nil {
			writeUnauthorized(w, "token inválido ou expirado")
			return
		}

		ctx := WithClaims(r.Context(), claims)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	response.WriteJSON(w, http.StatusUnauthorized, response.Fail(http.StatusUnauthorized, msg))
}

func WriteUnauthorized(w http.ResponseWriter) {
	writeUnauthorized(w, "token de autenticação não fornecido")
}

// PublicPaths builds a predicate that matches the given paths exactly, plus
// any path ending in "/" as a prefix. It is used for both the public path set
// and the optional-auth path set.
func PublicPaths(paths ...string) PublicPredicate {
	var exact []string
	var prefixes []string
	for _, p := range paths {
		if strings.HasSuffix(p, "/") {
			prefixes = append(prefixes, p)
		} else {
			exact = append(exact, p)
		}
	}
	exactSet := make(map[string]bool, len(exact))
	for _, p := range exact {
		exactSet[p] = true
	}

	return func(r *http.Request) bool {
		if exactSet[r.URL.Path] {
			return true
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				return true
			}
		}
		return strings.HasPrefix(r.URL.Path, "/swagger/")
	}
}
