package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangePassword(t *testing.T) {
	mux := newTestMux(t)

	do(t, mux, http.MethodPost, "/v1/auth/signup", `{"username":"pwuser","password":"password123","email":"pwuser@example.com"}`)
	loginRec := do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"pwuser","password":"password123"}`)

	var accessCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == "mm_at" {
			accessCookie = c
			break
		}
	}
	require.NotNil(t, accessCookie)

	change := func(body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/auth/change-password", strings.NewReader(body))
		req.AddCookie(accessCookie)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	t.Run("requires authentication", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/auth/change-password", strings.NewReader(`{"current_password":"password123","new_password":"new-password","confirm_new_password":"new-password"}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		rec := change(`{invalid json}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects mismatched confirmation", func(t *testing.T) {
		rec := change(`{"current_password":"password123","new_password":"new-password","confirm_new_password":"different"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects an incorrect current password", func(t *testing.T) {
		rec := change(`{"current_password":"wrong","new_password":"new-password","confirm_new_password":"new-password"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("changes the password", func(t *testing.T) {
		rec := change(`{"current_password":"password123","new_password":"new-password","confirm_new_password":"new-password"}`)
		require.Equal(t, http.StatusOK, rec.Code)

		assert.Equal(t, http.StatusUnauthorized,
			do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"pwuser","password":"password123"}`).Code)
		assert.Equal(t, http.StatusOK,
			do(t, mux, http.MethodPost, "/v1/auth/login", `{"username":"pwuser","password":"new-password"}`).Code)
	})
}
