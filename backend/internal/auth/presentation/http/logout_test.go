package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogout(t *testing.T) {
	mux := newTestMux(t)

	// Create account and login to get cookies
	do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"logoutuser","password":"password123","email":"logoutuser@example.com"}`)
	loginRec := do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"logoutuser","password":"password123"}`)

	cookies := loginRec.Result().Cookies()
	var accessCookie, refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "mm_at" {
			accessCookie = c
		}
		if c.Name == "mm_rt" {
			refreshCookie = c
		}
	}
	require.NotNil(t, accessCookie)
	require.NotNil(t, refreshCookie)

	t.Run("logout clears cookies", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/auth/logout", nil)
		req.AddCookie(accessCookie)
		req.AddCookie(refreshCookie)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		// Verify cookies are cleared (MaxAge = -1)
		cookies := rec.Result().Cookies()
		var clearedAccess, clearedRefresh *http.Cookie
		for _, c := range cookies {
			if c.Name == "mm_at" {
				clearedAccess = c
			}
			if c.Name == "mm_rt" {
				clearedRefresh = c
			}
		}
		require.NotNil(t, clearedAccess)
		require.NotNil(t, clearedRefresh)
		assert.Equal(t, -1, clearedAccess.MaxAge)
		assert.Equal(t, -1, clearedRefresh.MaxAge)
		assert.Empty(t, clearedAccess.Value)
		assert.Empty(t, clearedRefresh.Value)
	})

	t.Run("logout clears cookies without an access token", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/auth/logout", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, -1, findCookie(t, rec, "mm_at").MaxAge)
		assert.Equal(t, -1, findCookie(t, rec, "mm_rt").MaxAge)
	})
}
