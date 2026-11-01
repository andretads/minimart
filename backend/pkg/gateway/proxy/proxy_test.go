package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyPass(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "ok")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hi"))
	}))
	defer ts.Close()

	p, err := New(ts.URL, nil)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/test", nil)
	rr := httptest.NewRecorder()
	p.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "hi", rr.Body.String())
	require.Equal(t, "ok", rr.Header().Get("X-Upstream"))
}

func TestProxyPathJoining(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec // test helper writes path for verification; not user-facing
		_, _ = w.Write([]byte(r.URL.Path))
	}))
	defer ts.Close()

	p, err := New(ts.URL, nil)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/orders", nil)
	rr := httptest.NewRecorder()
	p.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	// Path should be joined: upstream path (empty) + incoming path
	assert.Equal(t, "/v1/orders", rr.Body.String())
}

func TestProxyQueryPreservation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec // test helper writes query for verification; not user-facing
		_, _ = w.Write([]byte(r.URL.RawQuery))
	}))
	defer ts.Close()

	p, err := New(ts.URL, nil)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/test?foo=bar&baz=qux", nil)
	rr := httptest.NewRecorder()
	p.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "foo=bar&baz=qux", rr.Body.String())
}

func TestProxyHostRewrite(t *testing.T) {
	var receivedHost string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHost = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p, err := New(ts.URL, nil)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/test", nil)
	req.Host = "original-host.example.com"
	rr := httptest.NewRecorder()
	p.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	// Host should be rewritten to upstream host (from ts.URL)
	expectedHost := ts.URL[len("http://"):]
	assert.Equal(t, expectedHost, receivedHost)
}

func TestProxyXForwardedHeaders(t *testing.T) {
	var receivedXFF, receivedXFP string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedXFF = r.Header.Get("X-Forwarded-For")
		receivedXFP = r.Header.Get("X-Forwarded-Proto")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p, err := New(ts.URL, nil)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/test", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	rr := httptest.NewRecorder()
	p.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	// X-Forwarded-For should be set from RemoteAddr
	assert.Equal(t, "192.0.2.1", receivedXFF)
	// X-Forwarded-Proto should be http for non-TLS
	assert.Equal(t, "http", receivedXFP)
}

func TestProxyModifyRequestDeletesXForwardedHost(t *testing.T) {
	var receivedXFH string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedXFH = r.Header.Get("X-Forwarded-Host")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Gateway deletes X-Forwarded-Host
	p, err := New(ts.URL, func(r *http.Request) {
		r.Header.Del("X-Forwarded-Host")
	})
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/test", nil)
	rr := httptest.NewRecorder()
	p.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	// X-Forwarded-Host should be deleted by modifier
	assert.Empty(t, receivedXFH)
}
