package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePinger stands in for a dependency. Set err for a failing check, or block to
// simulate a dependency that never answers.
type fakePinger struct {
	err   error
	block bool
}

func (f fakePinger) Ping(ctx context.Context) error {
	if f.block {
		<-ctx.Done()
		return ctx.Err()
	}
	return f.err
}

func get(t *testing.T, h *Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestLiveness(t *testing.T) {
	rec := get(t, New(nil), "/livez")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, map[string]string{"status": "alive"}, body)
}

func TestReadinessHealthy(t *testing.T) {
	h := New(map[string]Pinger{
		"read":  fakePinger{},
		"write": fakePinger{},
	})
	rec := get(t, h, "/readyz")

	require.Equal(t, http.StatusOK, rec.Code)

	var resp Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "healthy", resp.Status)
	assert.False(t, resp.Time.IsZero())
	require.Len(t, resp.Checks, 2)
	for name, check := range resp.Checks {
		assert.Equal(t, "up", check.Status, "check %q", name)
		assert.NotEmpty(t, check.Latency, "check %q", name)
		assert.Empty(t, check.Error, "check %q", name)
	}
}

func TestReadinessUnhealthyWhenAnyCheckFails(t *testing.T) {
	h := New(map[string]Pinger{
		"read":  fakePinger{},
		"write": fakePinger{err: errors.New("connection refused")},
	})
	rec := get(t, h, "/readyz")

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var resp Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "unhealthy", resp.Status)
	require.Len(t, resp.Checks, 2)
	assert.Equal(t, "down", resp.Checks["write"].Status)
	assert.Contains(t, resp.Checks["write"].Error, "connection refused")
	assert.Equal(t, "up", resp.Checks["read"].Status)
}

func TestReadinessTimeout(t *testing.T) {
	h := New(map[string]Pinger{"db": fakePinger{block: true}})
	h.timeout = 10 * time.Millisecond

	rec := get(t, h, "/readyz")

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var resp Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "unhealthy", resp.Status)
	assert.Contains(t, resp.Checks["db"].Error, "context deadline exceeded")
}
