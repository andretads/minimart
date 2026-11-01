package logger

import (
	"log/slog"
	"net/http"
	"time"
)

// requestIDHeader carries the correlation ID between the client, the proxy, and
// this service. An inbound value is honored so a request can be traced across
// hops; otherwise one is generated.
const requestIDHeader = "X-Request-ID"

// quietRoutes are high-frequency operational endpoints (Prometheus scrapes and
// orchestrator probes). Their access logs go to debug so they do not drown the
// request log at normal levels.
var quietRoutes = map[string]bool{
	"GET /metrics": true,
	"GET /livez":   true,
	"GET /readyz":  true,
}

// statusWriter captures the response status and byte count so the middleware can
// log them even when the handler never calls WriteHeader (implicit 200).
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// HTTPMiddleware assigns every request a correlation ID, exposes a request-scoped
// logger through the context, and writes one JSON access line when the request
// finishes. The line's level follows the response status (5xx error, 4xx warn,
// otherwise info), and operational routes are demoted to debug.
//
// It must wrap the mux outermost: it reads r.Pattern after routing for the
// bounded route template (never the raw, high-cardinality path), and its
// statusWriter observes the final status written by inner middleware.
func HTTPMiddleware(l *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(requestIDHeader, id)

		ctx := WithRequestID(r.Context(), id)
		ctx = WithLogger(ctx, l.With("request_id", id))
		r = r.WithContext(ctx)

		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)

		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		endpoint := r.Pattern
		if endpoint == "" {
			endpoint = "other"
		}

		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}
		if quietRoutes[r.Pattern] {
			level = slog.LevelDebug
		}

		l.LogAttrs(ctx, level, "http request",
			slog.String("request_id", id),
			slog.String("method", r.Method),
			slog.String("endpoint", endpoint),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.Int("bytes", sw.bytes),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
		)
	})
}
