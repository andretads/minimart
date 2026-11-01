package http

import (
	"encoding/json"
	"minimart/internal/customer/application"
	"minimart/internal/customer/infrastructure/memory"
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
	svc := application.NewService(repo, metrics.Noop{})
	mux := http.NewServeMux()
	NewHandlers(svc).Register(mux)
	return mux, repo
}

func do(t *testing.T, mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return doWithBody(t, mux, method, path, body)
}

func doWithBody(t *testing.T, mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(t.Context(), method, path, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

func doWithHeaders(t *testing.T, mux *http.ServeMux, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(t.Context(), method, path, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

func assertJSONResponse(t *testing.T, rec *httptest.ResponseRecorder, expectedStatus int, expectedSuccess bool) envelope {
	t.Helper()
	require.Equal(t, expectedStatus, rec.Code)
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	require.Equal(t, expectedSuccess, env.Success)
	return env
}

func buildCreateCustomerRequest(t *testing.T, userID, displayName string) string {
	t.Helper()
	req := createCustomerRequest{UserID: userID, DisplayName: displayName}
	b, err := json.Marshal(req)
	require.NoError(t, err)
	return string(b)
}

func buildUpdateCustomerRequest(t *testing.T, displayName string, version int) string {
	t.Helper()
	req := updateCustomerRequest{DisplayName: displayName, Version: version}
	b, err := json.Marshal(req)
	require.NoError(t, err)
	return string(b)
}
