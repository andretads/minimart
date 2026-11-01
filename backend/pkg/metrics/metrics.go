// Package metrics records MiniMart's operational and business signals for
// Prometheus scraping. The rest of the system depends only on the Metrics
// interface, never on client_golang - the same port/adapter split used for
// kafka.Publisher. The concrete PrometheusMetrics is built once in cmd and
// injected down the module boot.
//
// The client_golang implementation is split one file per subsystem
// (prom_http.go, prom_order.go, ...): each file owns its sub-struct, its
// collector definitions, and its recording methods.
package metrics

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// namespace prefixes every metric so the series are unmistakable in a shared
// Prometheus: minimart_http_requests_total, minimart_order_placed_total, ...
const namespace = "minimart"

// Result label values. Keeping them here stops callers from inventing spellings
// that would silently split one series into several.
const (
	StatusSuccess = "success"
	StatusError   = "error"
)

// Metrics is the recording port. HTTP middleware, application services, and the
// Kafka wrapper depend on this interface, so tests can inject Noop and no layer
// couples to client_golang.
type Metrics interface {
	IncRequest(endpoint, method, status string)
	ObserveLatency(endpoint, method string, d time.Duration)
	IncInFlight()
	DecInFlight()

	IncOrderPlaced(status string)
	ObserveOrderAmount(cents float64)
	IncOrderCancelled(status string)
	IncOrderPaymentApplied(status string)
	IncProductCreated(status string)
	IncProductUpdated(status string)
	IncStockReserved(status string)
	IncStockReleased(status string)
	IncNotificationSent(status string)
	IncNotificationPreferencesUpdated(status string)
	IncNotificationSuppressed(notificationType string)

	IncKafkaPublished(topic, status string)
	ObserveKafkaPublishLatency(topic string, d time.Duration)
	IncKafkaConsumed(topic, group, status string)
	ObserveKafkaConsumeLatency(topic, group string, d time.Duration)
	IncKafkaRetried(topic, group string)
	IncKafkaDeadLettered(topic, group, disposition string)
	IncKafkaConsumerRestart(topic, group string)
	SetKafkaConsumerLag(topic, group string, lag int64)
	IncKafkaBackpressureWait(topic, group string)
	SetKafkaConsumerPaused(topic, group string, paused bool)
	IncKafkaDLQReplayParked(topic string)

	IncRateLimited(scope, result string)
	ObserveRateLimitRemaining(scope string, remaining int)
	IncRedisCommand(command, status string)
	ObserveRedisCommandLatency(command string, d time.Duration)
	IncIdempotency(scope, result string)
	IncPayment(status string)
	ObservePaymentAmount(cents float64)
	IncCustomerEnsured(status string)
	IncCustomerCreated(status string)
	IncCustomerUpdated(status string)
	IncCustomerDeactivated(status string)

	IncCategoryCreated(status string)
	IncCategoryUpdated(status string)
	IncCategoryDeleted(status string)

	IncReviewAdded(status string)
	IncReviewModerated(decision string)

	IncCartCreated(status string)
	IncCartRetrieved(status string)
	IncCartItemAdded(status string)
	IncCartItemUpdated(status string)
	IncCartItemRemoved(status string)
	IncCartCleared(status string)
	IncCartCheckout(status string)
	IncCartConsumed(status string)

	IncWishlistCreated(status string)
	IncWishlistRetrieved(status string)
	IncWishlistItemAdded(status string)
	IncWishlistItemRemoved(status string)
	IncWishlistCleared(status string)

	IncShippingMethodsList(status string)
	IncShippingMethodGet(status string)
	IncShippingMethodCreate(status string)
	IncShippingMethodUpdate(status string)

	IncShippingZonesList(status string)
	IncShippingZoneGet(status string)
	IncShippingZoneCreate(status string)
	IncShippingZoneUpdate(status string)

	IncShippingRatesList(status string)
	IncShippingRateGet(status string)
	IncShippingRateCreate(status string)
	IncShippingRateUpdate(status string)

	IncShippingQuoteCreate(status string)
	IncShippingQuoteGet(status string)
	IncShippingQuoteValidate(status string)
	IncShippingQuotePrice(status string)
	IncShippingQuotesExpire(status string)

	IncShipmentCreate(status string)
	IncShipmentCreateFromOrderPaid(status string)
	IncShipmentGet(status string)
	IncShipmentList(status string)

	IncShipmentProcess(status string)
	IncShipmentDispatch(status string)
	IncShipmentDeliver(status string)

	IncTrackingEventCreate(status string)
	IncTrackingEventList(status string)

	IncAddressCreated(status string)
	IncAddressFetched(status string)
	IncAddressListed(status string)
	IncAddressUpdated(status string)
	IncAddressDeleted(status string)
	IncAddressDefaultSet(status string)

	IncAvatarUploaded(status string)
	IncAvatarFetched(status string)

	IncAccountCreated(status string)
	IncAuthenticated(status string)
	IncPasswordChanged(status string)
	IncTokenRefreshed(status string)

	IncRoleAssigned(status string)
	IncPermissionsLoaded(status string)
	IncAuthzDenied(status string)
}

// PrometheusMetrics is the client_golang implementation of Metrics: one embedded
// sub-struct per subsystem, each owning its collectors and recording methods.
// Every collector is registered on the registry passed to newPrometheusMetrics.
type PrometheusMetrics struct {
	httpMetrics
	orderMetrics
	catalogMetrics
	notificationMetrics
	kafkaMetrics
	platformMetrics
	paymentMetrics
	customerMetrics
	categoryMetrics
	cartMetrics
	wishlistMetrics
	addressMetrics
	shippingMetrics
	avatarMetrics
	authMetrics
}

var _ Metrics = (*PrometheusMetrics)(nil)

// NewPrometheusMetrics builds the collectors and registers them on the default
// registry (which already carries the process and Go runtime collectors), then
// returns the recorder. pools maps a name ("read", "write") to a pool; each
// contributes a labelled db_pool_* gauge. The map may be empty.
func NewPrometheusMetrics(pools map[string]*pgxpool.Pool) *PrometheusMetrics {
	return newPrometheusMetrics(prometheus.DefaultRegisterer, pools)
}

// newPrometheusMetrics is the test seam: passing a fresh registry keeps repeated
// construction in tests from panicking on duplicate registration.
func newPrometheusMetrics(reg prometheus.Registerer, pools map[string]*pgxpool.Pool) *PrometheusMetrics {
	m := &PrometheusMetrics{
		httpMetrics:         newHTTPMetrics(reg),
		orderMetrics:        newOrderMetrics(reg),
		catalogMetrics:      newCatalogMetrics(reg),
		notificationMetrics: newNotificationMetrics(reg),
		kafkaMetrics:        newKafkaMetrics(reg),
		platformMetrics:     newPlatformMetrics(reg),
		paymentMetrics:      newPaymentMetrics(reg),
		customerMetrics:     newCustomerMetrics(reg),
		categoryMetrics:     newCategoryMetrics(reg),
		cartMetrics:         newCartMetrics(reg),
		wishlistMetrics:     newWishlistMetrics(reg),
		addressMetrics:      newAddressMetrics(reg),
		shippingMetrics:     newShippingMetrics(reg),
		avatarMetrics:       newAvatarMetrics(reg),
		authMetrics:         newAuthMetrics(reg),
	}
	for name, pool := range pools {
		reg.MustRegister(poolGauges(name, pool)...)
	}
	return m
}

// poolGauges exposes pgxpool.Stat as GaugeFuncs - sampled at scrape time with no
// background goroutine and no custom Collector. The pool label keeps read and
// write pools apart.
func poolGauges(name string, pool *pgxpool.Pool) []prometheus.Collector {
	labels := prometheus.Labels{"pool": name}
	opts := func(metric, help string) prometheus.GaugeOpts {
		return prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "db_pool",
			Name:        metric,
			Help:        help,
			ConstLabels: labels,
		}
	}
	return []prometheus.Collector{
		// Alert: minimart_db_pool_acquired_connections{pool="write"} / minimart_db_pool_max_connections{pool="write"} > 0.9
		prometheus.NewGaugeFunc(opts("acquired_connections", "Number of connections currently in use."),
			func() float64 { return float64(pool.Stat().AcquiredConns()) }),
		prometheus.NewGaugeFunc(opts("idle_connections", "Number of idle connections in the pool."),
			func() float64 { return float64(pool.Stat().IdleConns()) }),
		prometheus.NewGaugeFunc(opts("total_connections", "Number of connections currently in the pool."),
			func() float64 { return float64(pool.Stat().TotalConns()) }),
		prometheus.NewGaugeFunc(opts("max_connections", "Maximum number of connections allowed in the pool."),
			func() float64 { return float64(pool.Stat().MaxConns()) }),
	}
}

// Handler serves the default registry, which carries the Go runtime and process
// collectors plus every collector registered above.
func Handler() http.Handler { return promhttp.Handler() }
