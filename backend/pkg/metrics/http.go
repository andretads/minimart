package metrics

import (
	"net/http"
	"strconv"
	"time"
)

// statusWriter captures the response status so the middleware can label the
// request even when a handler never calls WriteHeader (implicit 200).
type statusWriter struct {
	http.ResponseWriter
	status int
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
	return w.ResponseWriter.Write(b)
}

// HTTPMiddleware wraps the whole mux and records the golden signals for every
// request. It reads r.Pattern after the mux has routed - Go sets it on the same
// request value the mux received - so the endpoint label is the bounded route
// template ("GET /v1/products/{id}"), never a high-cardinality raw path.
func HTTPMiddleware(m Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m.IncInFlight()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		m.DecInFlight()

		endpoint := r.Pattern
		if endpoint == "" {
			endpoint = "other"
		}
		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		m.IncRequest(endpoint, r.Method, strconv.Itoa(status))
		m.ObserveLatency(endpoint, r.Method, time.Since(start))
	})
}
