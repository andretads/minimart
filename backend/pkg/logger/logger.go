// Package logger builds the process's structured logger and the HTTP middleware
// that logs every request. Logs are emitted as one JSON object per line on stdout
// (the format log collectors such as Splunk ingest directly), with the service,
// version, and environment attached to every record.
//
// It wraps the standard library's log/slog, so there are no third-party logging
// dependencies and the output stays compatible with the wider ecosystem.
package logger

import (
	"cmp"
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// loggerKey is the context key under which a request-scoped logger is stored.
type loggerKey struct{}

// Config configures the process logger. Zero values are filled with sane
// defaults: JSON on stdout at info level, tagged service "minimart".
type Config struct {
	Level     slog.Level // minimum level; zero is LevelInfo
	Format    string     // "json" (default) or "text"
	Output    io.Writer  // defaults to os.Stdout
	Service   string     // defaults to "minimart"
	Version   string     // build version, attached to every record
	Env       string     // deployment environment, attached to every record
	AddSource bool       // include file:line source, off by default
}

// New builds a logger from cfg without installing it as the process default.
func New(cfg Config) *slog.Logger {
	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}
	opts := &slog.HandlerOptions{
		Level:       cfg.Level,
		AddSource:   cfg.AddSource,
		ReplaceAttr: replaceAttr,
	}

	var handler slog.Handler
	if strings.EqualFold(cfg.Format, "text") {
		handler = slog.NewTextHandler(out, opts)
	} else {
		handler = slog.NewJSONHandler(out, opts)
	}

	l := slog.New(handler)

	attrs := make([]any, 0, 6)
	if service := cmp.Or(cfg.Service, "minimart"); service != "" {
		attrs = append(attrs, "service", service)
	}
	if cfg.Version != "" {
		attrs = append(attrs, "version", cfg.Version)
	}
	if cfg.Env != "" {
		attrs = append(attrs, "env", cfg.Env)
	}
	if len(attrs) > 0 {
		l = l.With(attrs...)
	}
	return l
}

// Setup builds the logger and installs it as slog's default, so every package
// that logs through slog (or the watermill bridge) shares one output pipeline.
func Setup(cfg Config) *slog.Logger {
	l := New(cfg)
	slog.SetDefault(l)
	return l
}

// ParseLevel maps a case-insensitive level name to a slog.Level. Unknown values
// fall back to info, so a typo in configuration cannot silence the logs.
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithLogger returns a context carrying l. HTTP middleware uses it to give every
// log line written while handling a request the same request_id.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// FromContext returns the logger stored in ctx, or slog's default when the
// context carries none.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

// replaceAttr normalizes attributes for log tooling: timestamps in UTC and
// durations as milliseconds under a "<key>_ms" name (slog otherwise writes raw
// nanosecond integers).
func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindTime:
		return slog.Time(a.Key, a.Value.Time().UTC())
	case slog.KindDuration:
		return slog.Float64(a.Key+"_ms", float64(a.Value.Duration())/float64(time.Millisecond))
	default:
		return a
	}
}
