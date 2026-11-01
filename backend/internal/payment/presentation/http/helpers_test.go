package http

import (
	"context"
	"encoding/json"
	"minimart/internal/payment/application"
	"minimart/internal/payment/infrastructure/memory"
	"minimart/internal/payment/infrastructure/provider/fake"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

type stubIdempotency struct {
	records map[string]idempotency.Record
}

func newStubIdempotency() *stubIdempotency {
	return &stubIdempotency{records: map[string]idempotency.Record{}}
}

func (s *stubIdempotency) Get(_ context.Context, key string) (idempotency.Record, bool, error) {
	r, ok := s.records[key]
	return r, ok, nil
}

func (s *stubIdempotency) Reserve(_ context.Context, key, fp string, _ time.Duration) (bool, error) {
	if _, ok := s.records[key]; ok {
		return false, nil
	}
	s.records[key] = idempotency.Record{Fingerprint: fp}
	return true, nil
}

func (s *stubIdempotency) Complete(_ context.Context, key, fp, resourceID string, _ time.Duration) error {
	s.records[key] = idempotency.Record{Fingerprint: fp, ResourceID: resourceID}
	return nil
}

func (s *stubIdempotency) Release(_ context.Context, key, _ string) error {
	delete(s.records, key)
	return nil
}

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newDefaultMux(t *testing.T) *http.ServeMux {
	t.Helper()
	provider := fake.New(fake.Config{DeclineAmountCents: 400, TimeoutAmountCents: 500})
	svc := application.NewService(memory.New(), nil, newStubIdempotency(), provider, metrics.Noop{}, 24*time.Hour)
	mux := http.NewServeMux()
	NewHandlers(svc).Register(mux)
	return mux
}

func do(t *testing.T, mux *http.ServeMux, method, path, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequestWithContext(t.Context(), method, path, nil)
	} else {
		r = httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	}
	if key != "" {
		r.Header.Set(idempotency.Header, key)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, r)
	return rec
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
