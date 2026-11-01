package http

import (
	"encoding/json"
	"minimart/internal/notification/application"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/memory"
	"minimart/pkg/auth"
	"minimart/pkg/metrics"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newTestMux(t *testing.T) (*http.ServeMux, *memory.Repository) {
	t.Helper()
	repo := memory.New()
	svc := application.NewService(repo, repo, repo, nil, nil, nil, nil, nil, metrics.Noop{}, application.RetryConfig{})
	mux := http.NewServeMux()
	NewHandlers(svc).Register(mux)
	return mux, repo
}

func do(t *testing.T, mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	r = r.WithContext(auth.WithClaims(r.Context(), &auth.Claims{UserID: "u-1"}))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

func doUnauth(t *testing.T, mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

func doAs(t *testing.T, mux *http.ServeMux, method, path, body, userID string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(t.Context(), method, path, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	}
	if userID != "" {
		r = r.WithContext(auth.WithClaims(r.Context(), &auth.Claims{UserID: userID}))
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

func seedNotification(t *testing.T, repo *memory.Repository, id, orderID, userID string) domain.Notification {
	t.Helper()
	n, err := domain.NewOrderConfirmation(id, orderID, userID, 1250)
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), n))
	return n
}
