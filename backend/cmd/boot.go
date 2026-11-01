package cmd

import (
	"context"
	"fmt"
	"minimart/conf"
	"minimart/internal/address"
	"minimart/internal/auth"
	"minimart/internal/catalog"
	"minimart/internal/category"
	"minimart/internal/customer"
	"minimart/internal/notification"
	"minimart/internal/notification/infrastructure/push"
	"minimart/internal/notification/infrastructure/sms"
	"minimart/internal/order"
	"minimart/internal/payment"
	"minimart/internal/shipping"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"minimart/pkg/module"
	"minimart/pkg/module/contracts"
	"minimart/pkg/redis"
	"minimart/pkg/resilience"

	authpkg "minimart/pkg/auth"
	kafkapkg "minimart/pkg/kafka"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// infra owns the process-wide dependencies shared by the server and worker
// roles: the data stores, the Kafka producer, and Redis.
type infra struct {
	readPool  *pgxpool.Pool
	writePool *pgxpool.Pool
	mongo     *mongo.Client
	rec       metrics.Metrics
	producer  *kafkapkg.Producer
	rdb       *redis.Client
}

// newInfra connects every shared dependency. The caller must defer Close,
// which releases them in reverse acquisition order.
func newInfra(ctx context.Context) (*infra, error) {
	readPool, writePool, err := newPools(ctx)
	if err != nil {
		return nil, err
	}
	mongoClient, err := newMongo(ctx)
	if err != nil {
		writePool.Close()
		readPool.Close()
		return nil, err
	}
	rec := newMetrics(readPool, writePool)
	producer, err := kafkapkg.NewProducerWithAuth(conf.KafkaBrokers(), kafkapkg.Auth{
		Username:  conf.KafkaUsername(),
		Password:  conf.KafkaPassword(),
		Mechanism: conf.KafkaSASLMechanism(),
		EnableTLS: conf.KafkaTLSEnabled(),
	}, rec)
	if err != nil {
		_ = mongoClient.Disconnect(ctx)
		writePool.Close()
		readPool.Close()
		return nil, err
	}
	rdb, err := newRedis()
	if err != nil {
		_ = producer.Close()
		_ = mongoClient.Disconnect(ctx)
		writePool.Close()
		readPool.Close()
		return nil, err
	}
	rdb.AddHook(metrics.NewRedisHook(rec))
	return &infra{
		readPool:  readPool,
		writePool: writePool,
		mongo:     mongoClient,
		rec:       rec,
		producer:  producer,
		rdb:       rdb,
	}, nil
}

// Close releases every dependency in reverse acquisition order. It must run
// before the caller cancels ctx.
func (i *infra) Close(ctx context.Context) {
	_ = i.rdb.Close()
	_ = i.producer.Close()
	_ = i.mongo.Disconnect(ctx)
	i.readPool.Close()
	i.writePool.Close()
}

// registerPoolGauges exports this process's Redis pool statistics.
func (i *infra) registerPoolGauges() {
	for _, c := range metrics.RedisPoolCollectors(i.rdb.Client, conf.RedisPoolSize()) {
		prometheus.MustRegister(c)
	}
}

// registerBreakerGauge exports a Redis circuit breaker's state as the
// minimart_redis_breaker_state gauge (0 closed, 1 half-open, 2 open).
func registerBreakerGauge(b *resilience.Breaker) {
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace:   "minimart",
		Subsystem:   "redis",
		Name:        "breaker_state",
		ConstLabels: prometheus.Labels{"name": b.Name()},
		Help:        "Circuit breaker state: 0 closed, 1 half-open, 2 open.",
	}, func() float64 {
		switch b.State() {
		case resilience.StateHalfOpen:
			return 1
		case resilience.StateOpen:
			return 2
		default:
			return 0
		}
	}))
}

// moduleDeps carries the dependencies the module constructors need.
type moduleDeps struct {
	readPool, writePool *pgxpool.Pool
	mongoDB             *mongo.Database
	registry            *contracts.Contracts
	rec                 metrics.Metrics
	producer            *kafkapkg.Producer
	rdb                 *redis.Client
	idem                idempotency.Store
	tokens              *authpkg.TokenService
	cookieEncryption    *authpkg.CookieEncryption
	cookieConfig        authpkg.CookieConfig
}

// newSharedModules builds the nine modules both the server and the worker
// boot, in dependency order: category registers its contract before catalog
// builds its service; order needs catalog and customer; shipping needs
// customer and address. The auth module is returned separately because both
// roles drive it after boot (admin seeding, the authz store).
func newSharedModules(d moduleDeps) ([]module.Module, *auth.Module) {
	authModule := auth.NewModule(d.registry, d.readPool, d.writePool, d.tokens, d.cookieEncryption, d.cookieConfig, conf.AuthRememberMeRefreshTokenTTL(), d.rec)
	modules := []module.Module{
		category.NewModule(d.readPool, d.writePool, d.rec),
		catalog.NewModule(d.registry, d.readPool, d.writePool, d.rec, d.rdb.Client),
		customer.NewModule(d.readPool, d.writePool, d.rec),
		address.NewModule(d.readPool, d.writePool, d.rec),
		authModule,
		order.NewModule(d.registry, d.readPool, d.writePool, d.producer, d.rec, d.idem, conf.OrderIdempotencyTTL(), orderOutboxSettings()),
		payment.NewModule(d.registry, d.readPool, d.writePool, d.producer, d.rec, d.idem, newPaymentProvider(), conf.PaymentIdempotencyTTL(), paymentOutboxSettings()),
		shipping.NewModule(d.registry, d.readPool, d.writePool, d.producer, d.rec, d.idem, conf.ShippingIdempotencyTTL(), shippingOutboxSettings()),
		notification.NewModule(d.mongoDB, d.registry, newEmailSender(), sms.NewLog(), push.NewLog(), d.producer, d.rec),
	}
	return modules, authModule
}

// bootModules runs the boot phases shared by every role: Init, then
// RegisterContracts, then the registry verification that fails boot when a
// contract slot is missing.
func bootModules(ctx context.Context, modules []module.Module, registry *contracts.Contracts) error {
	for _, m := range modules {
		if err := m.Init(ctx); err != nil {
			return fmt.Errorf("init module %T: %w", m, err)
		}
	}
	for _, m := range modules {
		if err := m.RegisterContracts(ctx, registry); err != nil {
			return fmt.Errorf("register contracts %T: %w", m, err)
		}
	}
	return registry.Verify()
}
