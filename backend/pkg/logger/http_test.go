package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger(buf *bytes.Buffer, level slog.Level) *slog.Logger {
	return New(Config{Output: buf, Level: level})
}

func TestHTTPMiddlewareLogsRequest(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf, slog.LevelInfo)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/products/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/products/42", nil)
	req.Header.Set(requestIDHeader, "abc-123")
	rec := httptest.NewRecorder()
	HTTPMiddleware(l, mux).ServeHTTP(rec, req)

	assert.Equal(t, "abc-123", rec.Header().Get(requestIDHeader))

	m := decodeLine(t, &buf)
	assert.Equal(t, "INFO", m["level"])
	assert.Equal(t, "http request", m["msg"])
	assert.Equal(t, "abc-123", m["request_id"])
	assert.Equal(t, http.MethodGet, m["method"])
	assert.Equal(t, "GET /v1/products/{id}", m["endpoint"])
	assert.EqualValues(t, http.StatusOK, m["status"])
	assert.EqualValues(t, 2, m["bytes"])
	assert.NotNil(t, m["duration_ms"])
}

func TestHTTPMiddlewareGeneratesRequestID(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf, slog.LevelInfo)
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	HTTPMiddleware(l, next).ServeHTTP(rec, req)

	id := rec.Header().Get(requestIDHeader)
	require.NotEmpty(t, id)

	m := decodeLine(t, &buf)
	assert.Equal(t, id, m["request_id"])
	assert.Equal(t, "other", m["endpoint"], "no mux means no route pattern")
}

func TestHTTPMiddlewareLevelByStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   string
	}{
		{"server error", http.StatusInternalServerError, "ERROR"},
		{"client error", http.StatusBadRequest, "WARN"},
		{"success", http.StatusOK, "INFO"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := newTestLogger(&buf, slog.LevelInfo)
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			})

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			HTTPMiddleware(l, next).ServeHTTP(httptest.NewRecorder(), req)

			m := decodeLine(t, &buf)
			assert.Equal(t, tt.want, m["level"])
			assert.EqualValues(t, tt.status, m["status"])
		})
	}
}

func TestHTTPMiddlewareQuietRoutesLogAtDebug(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", func(http.ResponseWriter, *http.Request) {})

	var debugBuf bytes.Buffer
	debugLogger := newTestLogger(&debugBuf, slog.LevelDebug)
	HTTPMiddleware(debugLogger, mux).ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)
	assert.Equal(t, "DEBUG", decodeLine(t, &debugBuf)["level"])

	var infoBuf bytes.Buffer
	infoLogger := newTestLogger(&infoBuf, slog.LevelInfo)
	HTTPMiddleware(infoLogger, mux).ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)
	assert.Empty(t, infoBuf.String(), "scrapes are suppressed at info level")
}

func TestHTTPMiddlewareProvidesContextLogger(t *testing.T) {
	var buf bytes.Buffer
	l := newTestLogger(&buf, slog.LevelInfo)

	var got *slog.Logger
	mux := http.NewServeMux()
	mux.HandleFunc("GET /x", func(_ http.ResponseWriter, r *http.Request) {
		got = FromContext(r.Context())
		assert.Equal(t, "req-1", RequestIDFromContext(r.Context()))
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", nil)
	req.Header.Set(requestIDHeader, "req-1")
	HTTPMiddleware(l, mux).ServeHTTP(httptest.NewRecorder(), req)

	require.NotNil(t, got)
	got.Info("inside handler")

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 2)
	var inner map[string]any
	require.NoError(t, json.Unmarshal(lines[1], &inner))
	assert.Equal(t, "req-1", inner["request_id"])
	assert.Equal(t, "inside handler", inner["msg"])
}
