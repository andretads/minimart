package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// newTestMetrics builds the real collectors on a throwaway registry, so tests
// never touch the default registry and can run in any order.
func newTestMetrics() *PrometheusMetrics {
	return newPrometheusMetrics(prometheus.NewRegistry(), nil)
}

// default-registry metrics may only be registered once per process; sync.OnceValue
// keeps TestHandlerExposesBusinessAndRuntimeMetrics re-runnable under -count=N.
var defaultRegistryMet = sync.OnceValue(func() *PrometheusMetrics {
	return NewPrometheusMetrics(nil)
})

func defaultRegistryMetrics() *PrometheusMetrics {
	return defaultRegistryMet()
}

func TestBusinessCounters(t *testing.T) {
	m := newTestMetrics()

	m.IncOrderPlaced(StatusSuccess)
	m.IncOrderPlaced(StatusSuccess)
	m.IncOrderPlaced(StatusError)
	m.ObserveOrderAmount(1250)
	m.IncProductCreated(StatusSuccess)
	m.IncProductUpdated(StatusSuccess)
	m.IncNotificationSent(StatusSuccess)
	m.IncCategoryCreated(StatusSuccess)
	m.IncCategoryUpdated(StatusSuccess)
	m.IncCategoryDeleted(StatusSuccess)
	m.IncAvatarUploaded(StatusSuccess)
	m.IncAvatarFetched(StatusSuccess)

	tests := []struct {
		name      string
		collector prometheus.Collector
		want      float64
	}{
		{"order_placed success", m.orderPlaced.WithLabelValues(StatusSuccess), 2},
		{"order_placed error", m.orderPlaced.WithLabelValues(StatusError), 1},
		{"product_created success", m.productCreated.WithLabelValues(StatusSuccess), 1},
		{"product_updated success", m.productUpdated.WithLabelValues(StatusSuccess), 1},
		{"notification_sent success", m.notificationSent.WithLabelValues(StatusSuccess), 1},
		{"category_created success", m.categoryCreated.WithLabelValues(StatusSuccess), 1},
		{"category_updated success", m.categoryUpdated.WithLabelValues(StatusSuccess), 1},
		{"category_deleted success", m.categoryDeleted.WithLabelValues(StatusSuccess), 1},
		{"avatar_uploaded success", m.avatarUploaded.WithLabelValues(StatusSuccess), 1},
		{"avatar_fetched success", m.avatarFetched.WithLabelValues(StatusSuccess), 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := testutil.ToFloat64(tt.collector); got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}

	if got := testutil.CollectAndCount(m.orderAmount); got != 1 {
		t.Fatalf("want 1 order amount series, got %d", got)
	}
}

func TestKafkaCounters(t *testing.T) {
	m := newTestMetrics()

	m.IncKafkaPublished("order-placed", StatusSuccess)
	m.IncKafkaPublished("order-placed", StatusError)
	m.IncKafkaConsumed("order-placed", "order-confirmation", StatusSuccess)

	if got := testutil.ToFloat64(m.kafkaPublished.WithLabelValues("order-placed", StatusSuccess)); got != 1 {
		t.Fatalf("want 1 published success, got %v", got)
	}
	if got := testutil.ToFloat64(m.kafkaPublished.WithLabelValues("order-placed", StatusError)); got != 1 {
		t.Fatalf("want 1 published error, got %v", got)
	}
	if got := testutil.ToFloat64(m.kafkaConsumed.WithLabelValues("order-placed", "order-confirmation", StatusSuccess)); got != 1 {
		t.Fatalf("want 1 consumed success, got %v", got)
	}
}

func TestHTTPMiddlewareLabelsRoutePattern(t *testing.T) {
	m := newTestMetrics()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/products/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := HTTPMiddleware(m, mux)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/products/42", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	// The route template, not the raw path, keeps cardinality bounded.
	got := testutil.ToFloat64(m.requests.WithLabelValues("GET /v1/products/{id}", http.MethodGet, "200"))
	if got != 1 {
		t.Fatalf("want 1 request recorded for route pattern, got %v", got)
	}
	if n := testutil.CollectAndCount(m.inFlight); n != 1 {
		t.Fatalf("want in-flight gauge exposed, got %d series", n)
	}
}

func TestHandlerExposesBusinessAndRuntimeMetrics(t *testing.T) {
	// Registers on the default registry, which is exactly what cmd builds and
	// Handler() serves. Registered once per process via sync.Once so the test
	// stays re-runnable under -count=N.
	m := defaultRegistryMetrics()
	m.IncOrderPlaced(StatusSuccess) // CounterVecs only appear once they have a child.

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		"minimart_http_requests_in_flight",
		"minimart_order_placed_total",
		"go_goroutines",
		"process_start_time_seconds",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q", want)
		}
	}
}
