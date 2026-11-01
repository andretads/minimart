package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeLine unmarshals a single JSON log line into a map for assertions.
func decodeLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &m))
	return m
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{" warn ", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"", slog.LevelInfo},
		{"nonsense", slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseLevel(tt.in))
		})
	}
}

func TestNewEmitsJSONWithBaseAttrs(t *testing.T) {
	var buf bytes.Buffer
	l := New(Config{Output: &buf, Service: "minimart", Version: "v1.2.3", Env: "production"})

	l.Info("server starting", "addr", ":8080")

	m := decodeLine(t, &buf)
	assert.Equal(t, "INFO", m["level"])
	assert.Equal(t, "server starting", m["msg"])
	assert.Equal(t, "minimart", m["service"])
	assert.Equal(t, "v1.2.3", m["version"])
	assert.Equal(t, "production", m["env"])
	assert.Equal(t, ":8080", m["addr"])
	assert.NotEmpty(t, m["time"])
}

func TestNewTextFormat(t *testing.T) {
	var buf bytes.Buffer
	l := New(Config{Format: "text", Output: &buf})

	l.Info("hello")

	assert.Contains(t, buf.String(), "msg=hello")
	assert.NotContains(t, buf.String(), "{")
}

func TestLevelFilters(t *testing.T) {
	var buf bytes.Buffer
	l := New(Config{Level: slog.LevelWarn, Output: &buf})

	l.Debug("debug")
	l.Info("info")
	assert.Empty(t, buf.String())

	l.Warn("warn")
	assert.Contains(t, buf.String(), "warn")
}

func TestDurationRenderedAsMilliseconds(t *testing.T) {
	var buf bytes.Buffer
	l := New(Config{Output: &buf})

	l.Info("request", slog.Duration("latency", 1500*time.Microsecond))

	m := decodeLine(t, &buf)
	assert.InDelta(t, 1.5, m["latency_ms"], 0.0001)
	_, ok := m["latency"]
	assert.False(t, ok, "raw duration key should be replaced")
}

func TestTimeIsUTC(t *testing.T) {
	var buf bytes.Buffer
	l := New(Config{Output: &buf})

	l.Info("tick")

	m := decodeLine(t, &buf)
	assert.Contains(t, m["time"], "Z")
}

func TestFromContextFallsBackToDefault(t *testing.T) {
	assert.Same(t, slog.Default(), FromContext(t.Context()))
}

func TestWithLoggerRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	l := New(Config{Output: &buf})

	ctx := WithLogger(t.Context(), l)

	assert.Same(t, l, FromContext(ctx))
}

func TestRequestIDRoundTrip(t *testing.T) {
	ctx := WithRequestID(t.Context(), "abc-123")

	assert.Equal(t, "abc-123", RequestIDFromContext(ctx))
	assert.Empty(t, RequestIDFromContext(t.Context()))
}

func TestNewRequestIDIsRandom(t *testing.T) {
	a := newRequestID()
	b := newRequestID()

	assert.NotEmpty(t, a)
	assert.NotEqual(t, a, b)
}
