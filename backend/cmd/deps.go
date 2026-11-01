package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"minimart/conf"
	authapp "minimart/internal/auth/application"
	"minimart/internal/notification/domain"
	"minimart/internal/notification/infrastructure/email/log"
	"minimart/internal/notification/infrastructure/email/resend"
	orderoutbox "minimart/internal/order/infrastructure/outbox"
	paymentdomain "minimart/internal/payment/domain"
	paymentoutbox "minimart/internal/payment/infrastructure/outbox"
	paymentfake "minimart/internal/payment/infrastructure/provider/fake"
	shippingoutbox "minimart/internal/shipping/infrastructure/outbox"
	authpkg "minimart/pkg/auth"
	"minimart/pkg/authz"
	"minimart/pkg/idempotency"
	idempotencyredis "minimart/pkg/idempotency/redis"
	idempotencyresilient "minimart/pkg/idempotency/resilient"
	"minimart/pkg/metrics"
	"minimart/pkg/ratelimit"
	ratelimitredis "minimart/pkg/ratelimit/redis"
	ratelimitresilient "minimart/pkg/ratelimit/resilient"
	"minimart/pkg/redis"
	"minimart/pkg/resilience"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

func newPools(ctx context.Context) (readPool, writePool *pgxpool.Pool, err error) {
	writePool, err = pgxpool.New(ctx, conf.DatabaseURL())
	if err != nil {
		return nil, nil, fmt.Errorf("connect primary: %w", err)
	}
	readPool, err = pgxpool.New(ctx, conf.DatabaseReplicaURL())
	if err != nil {
		writePool.Close()
		return nil, nil, fmt.Errorf("connect replica: %w", err)
	}
	return readPool, writePool, nil
}

func newMongo(ctx context.Context) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(conf.MongoURL()))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("ping mongo: %w", err)
	}
	return client, nil
}

type mongoPinger struct{ client *mongo.Client }

func (p mongoPinger) Ping(ctx context.Context) error {
	return p.client.Ping(ctx, readpref.Primary())
}

func newMetrics(readPool, writePool *pgxpool.Pool) metrics.Metrics {
	return metrics.NewPrometheusMetrics(map[string]*pgxpool.Pool{
		"read":  readPool,
		"write": writePool,
	})
}

func orderOutboxSettings() orderoutbox.Settings {
	return orderoutbox.Settings{
		ProcessInterval:           conf.OutboxProcessInterval(),
		LockCheckerInterval:       conf.OutboxLockCheckerInterval(),
		CleanupWorkerInterval:     conf.OutboxCleanupInterval(),
		MaxLockTimeDuration:       conf.OutboxMaxLockDuration(),
		MessagesRetentionDuration: conf.OutboxRetention(),
		MaxSendAttempts:           conf.OutboxMaxAttempts(),
	}
}

func newRedis() (*redis.Client, error) {
	return redis.NewWithOptions(conf.RedisURL(), redis.Options{
		DialTimeout:  conf.RedisDialTimeout(),
		ReadTimeout:  conf.RedisReadTimeout(),
		WriteTimeout: conf.RedisWriteTimeout(),
		PoolTimeout:  conf.RedisPoolTimeout(),
		MaxRetries:   conf.RedisMaxRetries(),
		PoolSize:     conf.RedisPoolSize(),
	})
}

func newRedisBreaker(name string) *resilience.Breaker {
	return resilience.New(name, resilience.Config{
		FailureThreshold: conf.RedisBreakerFailureThreshold(),
		OpenDuration:     conf.RedisBreakerOpenDuration(),
		HalfOpenMax:      1,
		SuccessThreshold: conf.RedisBreakerHalfOpenSuccesses(),
	})
}

func newIdempotencyStore(client *redis.Client, breaker *resilience.Breaker) idempotency.Store {
	return idempotencyresilient.New(idempotencyredis.New(client.Client), breaker, conf.RedisReadTimeout())
}

func newRateLimiter(client *redis.Client, breaker *resilience.Breaker) ratelimit.Limiter {
	return ratelimitresilient.New(ratelimitredis.New(client.Client, conf.RateLimitBurst(), conf.RateLimitRefillTokens(), conf.RateLimitRefillInterval()), breaker, conf.RedisReadTimeout())
}

func newPaymentProvider() paymentdomain.Provider {
	return paymentfake.New(paymentfake.Config{
		DeclineAmountCents: conf.PaymentFakeDeclineAmountCents(),
		TimeoutAmountCents: conf.PaymentFakeTimeoutAmountCents(),
	})
}

func paymentOutboxSettings() paymentoutbox.Settings {
	return paymentoutbox.Settings{
		ProcessInterval:           conf.OutboxProcessInterval(),
		LockCheckerInterval:       conf.OutboxLockCheckerInterval(),
		CleanupWorkerInterval:     conf.OutboxCleanupInterval(),
		MaxLockTimeDuration:       conf.OutboxMaxLockDuration(),
		MessagesRetentionDuration: conf.OutboxRetention(),
		MaxSendAttempts:           conf.OutboxMaxAttempts(),
	}
}

func shippingOutboxSettings() shippingoutbox.Settings {
	return shippingoutbox.Settings{
		ProcessInterval:           conf.OutboxProcessInterval(),
		LockCheckerInterval:       conf.OutboxLockCheckerInterval(),
		CleanupWorkerInterval:     conf.OutboxCleanupInterval(),
		MaxLockTimeDuration:       conf.OutboxMaxLockDuration(),
		MessagesRetentionDuration: conf.OutboxRetention(),
		MaxSendAttempts:           conf.OutboxMaxAttempts(),
	}
}

func newCookieEncryption() *authpkg.CookieEncryption {
	key := conf.CookieEncryptionKey()
	enc, err := authpkg.NewCookieEncryption(key)
	if err != nil {
		panic(err)
	}
	return enc
}

func newCookieConfig() authpkg.CookieConfig {
	env := conf.Env()
	if env == "production" {
		domain := os.Getenv("COOKIE_DOMAIN")
		return authpkg.ProductionCookieConfig(conf.AuthAccessTokenTTL(), conf.AuthRefreshTokenTTL(), domain)
	}
	return authpkg.DefaultCookieConfig(conf.AuthAccessTokenTTL(), conf.AuthRefreshTokenTTL())
}

func newTokenService() *authpkg.TokenService {
	svc, err := authpkg.NewTokenService(conf.JWTSecret(), conf.AuthAccessTokenTTL(), conf.AuthRefreshTokenTTL())
	if err != nil {
		panic(err)
	}
	return svc
}

func newAuthzStore(authModule *authapp.Service, rdb *redis.Client) *authz.CachedStore {
	return authz.NewCachedStoreWithOptions(authModule, rdb, conf.RBACPermissionCacheTTL(), conf.RBACPermissionLocalCacheTTL(), slog.Default())
}

func newEmailSender() domain.EmailSender {
	if key := conf.ResendAPIKey(); key != "" {
		slog.Info("resend email sender configured", "from", conf.ResendFrom())
		return resend.New(key, conf.ResendFrom(), conf.ResendReplyTo())
	}
	slog.Warn("RESEND_API_KEY not set; using log fallback for emails")
	return log.New()
}
