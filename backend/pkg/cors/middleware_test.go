package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreflight(t *testing.T) {
	cfg := Config{Origins: []string{"*"}, Methods: "GET,POST", Headers: "*", ExposeHeaders: "X-Request-ID", Credentials: false, MaxAge: "3600"}
	h := Middleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/v1/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code)
	require.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
	require.NotEmpty(t, rr.Header().Get("Access-Control-Allow-Methods"))

	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Credentials"))
}
