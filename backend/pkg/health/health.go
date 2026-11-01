// Package health exposes the operational liveness and readiness endpoints that
// orchestrators (Kubernetes, load balancers) probe. Liveness says "the process is
// up"; readiness says "the process can serve traffic", which means every critical
// dependency - here the database pools - answers a ping.
package health

import (
	"context"
	"log/slog"
	"minimart/pkg/response"
	"net/http"
	"sort"
	"time"
)

// Status values reported in the JSON body. Kept as constants so callers and probes
// agree on exact spellings.
const (
	statusAlive     = "alive"
	statusHealthy   = "healthy"
	statusDegraded  = "degraded"
	statusUnhealthy = "unhealthy"
	statusUp        = "up"
	statusDown      = "down"
)

// defaultCheckTimeout bounds a readiness probe so one unresponsive dependency
// cannot hang the endpoint.
const defaultCheckTimeout = 2 * time.Second

// Pinger is the dependency this handler needs: anything that can verify it is
// reachable. *pgxpool.Pool satisfies it, and tests inject fakes so readiness can
// be exercised without a database.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Dependency describes one readiness check. Critical dependencies gate readiness:
// if any critical dependency is down, /readyz returns 503. Non-critical
// dependencies (such as Redis, an accelerator) are reported as "degraded" but
// keep the endpoint returning 200.
type Dependency struct {
	Name     string
	Pinger   Pinger
	Critical bool
}

// Response is the readiness body. Time is when the check ran; Checks holds one
// entry per dependency, keyed by the name the composition root gave it ("read",
// "write").
type Response struct {
	Time   time.Time        `json:"time"`
	Status string           `json:"status"`
	Checks map[string]Check `json:"checks,omitempty"`
}

// Check is the result of pinging one dependency.
type Check struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Handler serves /livez and /readyz. It holds a labelled set of dependencies
// rather than concrete pools, so the same handler works in the server and the
// worker.
type Handler struct {
	deps    []Dependency
	timeout time.Duration
}

// New builds a handler over the named checks. Every check is critical, preserving
// the original behavior. The map may be empty, in which case readiness is
// trivially healthy.
func New(checks map[string]Pinger) *Handler {
	deps := make([]Dependency, 0, len(checks))
	names := make([]string, 0, len(checks))
	for name := range checks {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		deps = append(deps, Dependency{Name: name, Pinger: checks[name], Critical: true})
	}
	return &Handler{deps: deps, timeout: defaultCheckTimeout}
}

// NewWithDependencies builds a handler from an explicit dependency list. It
// accepts either []health.Dependency or map[string]health.Dependency so callers
// can choose the most convenient form. An optional trailing timeout overrides
// the default 2s probe timeout.
func NewWithDependencies(deps any, timeout ...time.Duration) *Handler {
	h := &Handler{timeout: defaultCheckTimeout}
	if len(timeout) > 0 {
		h.timeout = timeout[0]
	}
	switch v := deps.(type) {
	case []Dependency:
		h.deps = append([]Dependency(nil), v...)
	case map[string]Dependency:
		names := make([]string, 0, len(v))
		for name := range v {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			d := v[name]
			if d.Name == "" {
				d.Name = name
			}
			h.deps = append(h.deps, d)
		}
	case map[string]Pinger:
		names := make([]string, 0, len(v))
		for name := range v {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			h.deps = append(h.deps, Dependency{Name: name, Pinger: v[name], Critical: true})
		}
	}
	return h
}

// NewWithDeps is an alias for NewWithDependencies.
func NewWithDeps(deps any, timeout ...time.Duration) *Handler {
	return NewWithDependencies(deps, timeout...)
}

// Register wires the health routes onto the shared mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /livez", h.Liveness)
	mux.HandleFunc("GET /readyz", h.Readiness)
}

// Liveness reports that the process is running. It touches no dependency: a
// dependency outage must not make the orchestrator restart a healthy process.
func (h *Handler) Liveness(w http.ResponseWriter, _ *http.Request) {
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": statusAlive})
}

// Readiness pings every dependency and reports the result. Critical failures
// yield 503; non-critical failures yield 200 with a "degraded" status.
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	checks := make(map[string]Check, len(h.deps))
	healthy := true
	degraded := false
	for _, dep := range h.deps {
		start := time.Now()
		err := dep.Pinger.Ping(ctx)
		latency := time.Since(start)

		if err != nil {
			checks[dep.Name] = Check{Status: statusDown, Error: err.Error()}
			slog.Error("readiness check failed", "check", dep.Name, "error", err)
			if dep.Critical {
				healthy = false
			} else {
				degraded = true
			}
			continue
		}
		checks[dep.Name] = Check{Status: statusUp, Latency: latency.String()}
	}

	resp := Response{Time: time.Now(), Status: statusHealthy, Checks: checks}
	status := http.StatusOK
	if !healthy {
		resp.Status = statusUnhealthy
		status = http.StatusServiceUnavailable
	} else if degraded {
		resp.Status = statusDegraded
	}
	response.WriteJSON(w, status, resp)
}
