// Package conf reads all configuration from environment variables in one place.
// Defaults target the local docker-compose stack, so the service runs with no
// .env file in development. In production every value comes from the environment.
package conf

import (
	"cmp"
	"encoding/base64"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

func env(key, def string) string {
	return cmp.Or(os.Getenv(key), def)
}

// splitCSV splits a comma-separated string, trimming whitespace from each part,
// and filtering out empty parts. Returns nil for empty input.
func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for part := range strings.SplitSeq(raw, ",") {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// envDuration parses a Go duration ("5s", "1m") from the environment, falling back
// to def when unset or unparseable.
func envDuration(key string, def time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return def
	}
	return d
}

// envInt parses an integer from the environment, falling back to def when unset
// or unparseable.
func envInt(key string, def int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}

// envBool parses a boolean from the environment, falling back to def when unset
// or unparseable.
func envBool(key string, def bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return def
	}
	return b
}

// DatabaseURL is the primary (read-write) Postgres DSN. Writes go here.
func DatabaseURL() string {
	return env("DATABASE_URL", "postgres://minimart:minimart@localhost:5432/minimart?sslmode=disable")
}

// DatabaseReplicaURL is the read-replica DSN. Reads go here. It falls back to the
// primary when unset - the common case in local development, where there is only
// one database.
func DatabaseReplicaURL() string {
	return cmp.Or(os.Getenv("DATABASE_REPLICA_URL"), DatabaseURL())
}

// MongoURL is the MongoDB connection string the notification domain persists to.
func MongoURL() string {
	return env("MONGO_URL", "mongodb://localhost:27017")
}

// MongoDatabase is the MongoDB database name the notification domain uses.
func MongoDatabase() string {
	return env("MONGO_DATABASE", "minimart")
}

// ServerHTTPAddr is the address the server role's HTTP API listens on.
func ServerHTTPAddr() string {
	return env("SERVER_HTTP_ADDR", ":8080")
}

// WorkerHTTPAddr is the address the worker role's small HTTP listener binds.
// The worker has no API; it serves /metrics plus the health probes, so it gets
// its own listener. The server exposes /metrics on ServerHTTPAddr instead.
func WorkerHTTPAddr() string {
	return env("WORKER_HTTP_ADDR", ":8081")
}

// Env identifies the deployment environment for logs or environment-specific
// behavior at the edges.
func Env() string {
	return env("ENV", "development")
}

// KafkaBrokers is the comma-separated list of Kafka bootstrap brokers.
func KafkaBrokers() []string {
	return splitCSV(env("KAFKA_BROKERS", "localhost:9092"))
}

// KafkaUsername is the SASL username for Kafka (Confluent Cloud). Empty disables auth.
func KafkaUsername() string {
	return os.Getenv("KAFKA_USERNAME")
}

// KafkaPassword is the SASL password for Kafka (Confluent Cloud).
func KafkaPassword() string {
	return os.Getenv("KAFKA_PASSWORD")
}

// KafkaSASLMechanism is the SASL mechanism ("plain", "scram-sha-256", "scram-sha-512").
func KafkaSASLMechanism() string {
	return env("KAFKA_SASL_MECHANISM", "plain")
}

// KafkaTLSEnabled forces TLS for Kafka connections. Auto-enabled when a username is set.
func KafkaTLSEnabled() bool {
	return envBool("KAFKA_TLS_ENABLED", false)
}

// KafkaConsumerMaxAttempts is the total number of handler calls per message
// before it is dead-lettered. One disables retries.
func KafkaConsumerMaxAttempts() int {
	return envInt("KAFKA_CONSUMER_MAX_ATTEMPTS", 5)
}

// KafkaConsumerRetryBaseDelay is the first wait before retrying a failed handler.
func KafkaConsumerRetryBaseDelay() time.Duration {
	return envDuration("KAFKA_CONSUMER_RETRY_BASE_DELAY", 500*time.Millisecond)
}

// KafkaConsumerRetryMaxDelay caps the exponential retry backoff.
func KafkaConsumerRetryMaxDelay() time.Duration {
	return envDuration("KAFKA_CONSUMER_RETRY_MAX_DELAY", 30*time.Second)
}

// KafkaConsumerRestartMinBackoff is the first wait before restarting a consumer
// that stopped unexpectedly.
func KafkaConsumerRestartMinBackoff() time.Duration {
	return envDuration("KAFKA_CONSUMER_RESTART_MIN_BACKOFF", time.Second)
}

// KafkaConsumerRestartMaxBackoff caps the consumer restart backoff.
func KafkaConsumerRestartMaxBackoff() time.Duration {
	return envDuration("KAFKA_CONSUMER_RESTART_MAX_BACKOFF", 30*time.Second)
}

// KafkaConsumerConcurrency is the number of supervised consumer replicas run per
// (topic, group) subscription. It multiplies consumption throughput; values
// below one are normalized to one.
func KafkaConsumerConcurrency() int {
	n := envInt("KAFKA_CONSUMER_CONCURRENCY", 1)
	return max(n, 1)
}

// KafkaConsumerPauseEnabled turns on the dependency-saturation pause gate.
func KafkaConsumerPauseEnabled() bool {
	return envBool("KAFKA_CONSUMER_PAUSE_ENABLED", false)
}

// KafkaConsumerPausePoolThreshold is the pool utilization above which consumption
// pauses. Values outside (0,1] fall back to 0.9.
func KafkaConsumerPausePoolThreshold() float64 {
	raw := os.Getenv("KAFKA_CONSUMER_PAUSE_POOL_THRESHOLD")
	if raw == "" {
		return 0.9
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || v <= 0 || v > 1 {
		return 0.9
	}
	return v
}

// KafkaConsumerPauseDuration is how long a pause lasts before consumption resumes.
func KafkaConsumerPauseDuration() time.Duration {
	return envDuration("KAFKA_CONSUMER_PAUSE_DURATION", 5*time.Second)
}

// KafkaConsumerPauseInterval is how often pool utilization is sampled.
func KafkaConsumerPauseInterval() time.Duration {
	return envDuration("KAFKA_CONSUMER_PAUSE_INTERVAL", time.Second)
}

// KafkaConsumerLagEnabled turns on the consumer-lag monitor.
func KafkaConsumerLagEnabled() bool {
	return envBool("KAFKA_CONSUMER_LAG_ENABLED", true)
}

// KafkaConsumerLagInterval is how often consumer lag is sampled.
func KafkaConsumerLagInterval() time.Duration {
	return envDuration("KAFKA_CONSUMER_LAG_INTERVAL", 30*time.Second)
}

// RedisURL is the Redis connection string backing the payment domain's
// idempotency-key cache and the global HTTP rate limiter.
func RedisURL() string {
	return env("REDIS_URL", "redis://localhost:6379/0")
}

// RedisRequiredForReadiness controls whether Redis is a critical readiness
// dependency. It defaults to false: Redis is an accelerator and an outage must
// not make the process report itself unavailable.
func RedisRequiredForReadiness() bool {
	return envBool("REDIS_REQUIRED_FOR_READINESS", false)
}

// RedisDialTimeout bounds establishing a Redis connection.
func RedisDialTimeout() time.Duration {
	return envDuration("REDIS_DIAL_TIMEOUT", 2*time.Second)
}

// RedisReadTimeout bounds a Redis read command.
func RedisReadTimeout() time.Duration {
	return envDuration("REDIS_READ_TIMEOUT", 200*time.Millisecond)
}

// RedisWriteTimeout bounds a Redis write command.
func RedisWriteTimeout() time.Duration {
	return envDuration("REDIS_WRITE_TIMEOUT", 200*time.Millisecond)
}

// RedisPoolTimeout bounds waiting for a connection from the pool.
func RedisPoolTimeout() time.Duration {
	return envDuration("REDIS_POOL_TIMEOUT", 200*time.Millisecond)
}

// RedisMaxRetries is the number of retries go-redis performs for a failed
// command. Zero disables retries so a degraded Redis cannot amplify latency.
func RedisMaxRetries() int {
	return envInt("REDIS_MAX_RETRIES", 0)
}

// RedisPoolSize is the size of the Redis connection pool.
func RedisPoolSize() int {
	return envInt("REDIS_POOL_SIZE", 10)
}

// RedisBreakerFailureThreshold is the number of consecutive Redis failures
// before the circuit breaker opens.
func RedisBreakerFailureThreshold() int {
	return envInt("REDIS_BREAKER_FAILURE_THRESHOLD", 5)
}

// RedisBreakerOpenDuration is how long the Redis circuit breaker stays open.
func RedisBreakerOpenDuration() time.Duration {
	return envDuration("REDIS_BREAKER_OPEN_DURATION", 10*time.Second)
}

// RedisBreakerHalfOpenSuccesses is the number of successful probes required to
// close the Redis circuit breaker.
func RedisBreakerHalfOpenSuccesses() int {
	return envInt("REDIS_BREAKER_HALF_OPEN_SUCCESSES", 2)
}

func RateLimitBurst() int {
	return envInt("RATE_LIMIT_BURST", 10)
}

func RateLimitRefillTokens() int {
	return envInt("RATE_LIMIT_REFILL_TOKENS", 10)
}

func RateLimitRefillInterval() time.Duration {
	return envDuration("RATE_LIMIT_REFILL_INTERVAL", time.Second)
}

// PaymentIdempotencyTTL is how long a payment's idempotency key is remembered,
// both in Redis and for idempotent replay.
func PaymentIdempotencyTTL() time.Duration {
	return envDuration("PAYMENT_IDEMPOTENCY_TTL", 24*time.Hour)
}

// OrderIdempotencyTTL is how long an order's idempotency key is remembered, both
// in Redis and for idempotent replay.
func OrderIdempotencyTTL() time.Duration {
	return envDuration("ORDER_IDEMPOTENCY_TTL", 24*time.Hour)
}

// OrderAutoCancelInterval is how often the worker scans for unpaid placed orders.
func OrderAutoCancelInterval() time.Duration {
	return envDuration("ORDER_AUTO_CANCEL_INTERVAL", 15*time.Second)
}

// OrderAutoCancelBatchSize is the maximum number of orders cancelled per scan.
func OrderAutoCancelBatchSize() int {
	return envInt("ORDER_AUTO_CANCEL_BATCH_SIZE", 100)
}

// ShippingIdempotencyTTL is how long a shipping quote's idempotency key is remembered,
// both in Redis and for idempotent replay.
func ShippingIdempotencyTTL() time.Duration {
	return envDuration("SHIPPING_IDEMPOTENCY_TTL", 24*time.Hour)
}

// PaymentFakeDeclineAmountCents is the amount the fake provider declines.
// Zero disables the decline case.
func PaymentFakeDeclineAmountCents() int64 {
	return int64(envInt("PAYMENT_FAKE_DECLINE_AMOUNT_CENTS", 9999))
}

// PaymentFakeTimeoutAmountCents is the amount the fake provider simulates a
// timeout for, which the domain maps to the unknown state. Zero disables it.
func PaymentFakeTimeoutAmountCents() int64 {
	return int64(envInt("PAYMENT_FAKE_TIMEOUT_AMOUNT_CENTS", 7777))
}

// OutboxProcessInterval is how often the worker's outbox dispatcher polls for
// undelivered records and relays them to Kafka.
func OutboxProcessInterval() time.Duration {
	return envDuration("OUTBOX_PROCESS_INTERVAL", 5*time.Second)
}

// OutboxLockCheckerInterval is how often stale outbox locks (left by a crashed
// dispatcher) are reclaimed.
func OutboxLockCheckerInterval() time.Duration {
	return envDuration("OUTBOX_LOCK_CHECKER_INTERVAL", 30*time.Second)
}

// OutboxCleanupInterval is how often delivered outbox records are retired.
func OutboxCleanupInterval() time.Duration {
	return envDuration("OUTBOX_CLEANUP_INTERVAL", 5*time.Minute)
}

// OutboxMaxLockDuration is how long a dispatcher may hold a record lock before it
// is considered abandoned and reclaimed.
func OutboxMaxLockDuration() time.Duration {
	return envDuration("OUTBOX_MAX_LOCK_DURATION", time.Minute)
}

// OutboxRetention is how long delivered outbox records are kept before cleanup.
func OutboxRetention() time.Duration {
	return envDuration("OUTBOX_RETENTION", 24*time.Hour)
}

// OutboxMaxAttempts caps how many times a record is published before it is
// abandoned. Zero disables the cap (retry forever).
func OutboxMaxAttempts() int {
	return envInt("OUTBOX_MAX_ATTEMPTS", 5)
}

// LogLevel is the minimum level emitted by the process logger: "debug", "info",
// "warn", or "error".
func LogLevel() string {
	return env("LOG_LEVEL", "info")
}

// LogFormat selects the log encoding: "json" (default, for log collectors such
// as Splunk) or "text" for readable local development.
func LogFormat() string {
	return env("LOG_FORMAT", "json")
}

// JWTSecret is the secret used to sign JWT tokens.
func JWTSecret() string {
	return env("JWT_SECRET", "minimart-local-dev-secret-change-me")
}

// AuthAccessTokenTTL is the lifetime of access tokens.
func AuthAccessTokenTTL() time.Duration {
	return envDuration("AUTH_ACCESS_TOKEN_TTL", 15*time.Minute)
}

// AuthRefreshTokenTTL is the lifetime of refresh tokens.
func AuthRefreshTokenTTL() time.Duration {
	return envDuration("AUTH_REFRESH_TOKEN_TTL", 7*24*time.Hour)
}

// AuthRememberMeRefreshTokenTTL is the lifetime of refresh tokens when "Remember me" is checked.
func AuthRememberMeRefreshTokenTTL() time.Duration {
	return envDuration("AUTH_REMEMBER_ME_REFRESH_TOKEN_TTL", 30*24*time.Hour)
}

// AdminUsernames is a comma-separated list of usernames that should be granted
// the admin role at boot. Empty disables seeding.
func AdminUsernames() []string {
	return splitCSV(os.Getenv("ADMIN_USERNAMES"))
}

// RBACPermissionCacheTTL is how long a permission set is cached in Redis.
func RBACPermissionCacheTTL() time.Duration {
	return envDuration("RBAC_PERMISSIONS_CACHE_TTL", 5*time.Minute)
}

// RBACPermissionLocalCacheTTL is how long a permission set is cached in process
// memory. It bounds database fallback traffic during a Redis outage.
func RBACPermissionLocalCacheTTL() time.Duration {
	return envDuration("RBAC_PERMISSIONS_LOCAL_CACHE_TTL", time.Minute)
}

// PaymentReconcileInterval is how often the worker reconciles unknown payments.
func PaymentReconcileInterval() time.Duration {
	return envDuration("PAYMENT_RECONCILE_INTERVAL", time.Minute)
}

// PaymentReconcileAge is how old a pending/unknown payment must be before reconciliation.
func PaymentReconcileAge() time.Duration {
	return envDuration("PAYMENT_RECONCILE_AGE", 5*time.Minute)
}

// PaymentReconcileBatchSize is the maximum number of payments reconciled per run.
func PaymentReconcileBatchSize() int {
	return envInt("PAYMENT_RECONCILE_BATCH_SIZE", 100)
}

// GatewayHTTPAddr is the address the gateway listens on.
func GatewayHTTPAddr() string {
	return env("GATEWAY_HTTP_ADDR", ":8080")
}

// GatewayUpstream is the URL of the upstream server role the gateway proxies to.
func GatewayUpstream() string {
	return env("GATEWAY_UPSTREAM", "http://localhost:8080")
}

// GatewayTrustedProxies is a comma-separated list of trusted proxy CIDRs.
func GatewayTrustedProxies() []string {
	return splitCSV(os.Getenv("GATEWAY_TRUSTED_PROXIES"))
}

// GatewayCORSOrigins is the list of allowed CORS origins.
func GatewayCORSOrigins() []string {
	raw := os.Getenv("GATEWAY_CORS_ORIGINS")
	if raw == "" {
		return []string{"*"}
	}
	return splitCSV(raw)
}

// GatewayCORSMethods are allowed HTTP methods for CORS preflight.
func GatewayCORSMethods() string {
	return env("GATEWAY_CORS_METHODS", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
}

// GatewayCORSHeaders are allowed request headers for CORS preflight.
func GatewayCORSHeaders() string {
	return env("GATEWAY_CORS_HEADERS", "*")
}

// GatewayCORSExposeHeaders are headers exposed to the browser.
func GatewayCORSExposeHeaders() string {
	return env("GATEWAY_CORS_EXPOSE_HEADERS", "X-Request-ID")
}

// GatewayCORSCredentials indicates whether credentials are allowed.
func GatewayCORSCredentials() bool {
	return envBool("GATEWAY_CORS_CREDENTIALS", true)
}

// GatewayCORSMaxAge is the max age for preflight cache.
func GatewayCORSMaxAge() string {
	return env("GATEWAY_CORS_MAX_AGE", "3600")
}

// CookieEncryptionKey returns the base64-encoded 32-byte key for encrypting auth cookies.
// If not set, derives from JWT_SECRET (dev only). In production, must be set explicitly.
func CookieEncryptionKey() []byte {
	raw := os.Getenv("COOKIE_ENCRYPTION_KEY")
	if raw != "" {
		key, err := base64.RawURLEncoding.DecodeString(raw)
		if err == nil && len(key) == 32 {
			return key
		}
		slog.Warn("COOKIE_ENCRYPTION_KEY is not base64 for 32 bytes; falling back to the JWT_SECRET-derived key",
			"error", err, "decoded_bytes", len(key))
	}
	// Derive from JWT_SECRET as fallback (dev only)
	secret := JWTSecret()
	if len(secret) >= 32 {
		return []byte(secret[:32])
	}
	return append([]byte(secret), make([]byte, 32-len(secret))...)
}

// S3Endpoint is the S3-compatible storage endpoint (e.g., MinIO, floci).
// Empty means use AWS S3 default endpoint.
func S3Endpoint() string {
	return env("S3_ENDPOINT", "")
}

// S3PublicEndpoint is the browser-facing S3 endpoint for presigned URLs.
// If empty, defaults to S3Endpoint().
func S3PublicEndpoint() string {
	return env("S3_PUBLIC_ENDPOINT", S3Endpoint())
}

// S3Bucket is the S3 bucket name for product images.
func S3Bucket() string {
	return env("S3_BUCKET", "minimart")
}

// S3Region is the AWS region for S3.
func S3Region() string {
	return env("S3_REGION", "us-east-1")
}

// S3AccessKey is the S3 access key ID.
func S3AccessKey() string {
	return env("S3_ACCESS_KEY", "minioadmin")
}

// S3SecretKey is the S3 secret access key.
func S3SecretKey() string {
	return env("S3_SECRET_KEY", "minioadmin")
}

// S3UsePathStyle enables path-style addressing for S3 (required for MinIO/floci).
func S3UsePathStyle() bool {
	return envBool("S3_USE_PATH_STYLE", true)
}

// S3PresignTTL is the TTL for presigned URLs.
func S3PresignTTL() time.Duration {
	return envDuration("S3_PRESIGN_TTL", 1*time.Hour)
}

// S3MaxUploadSizeMB is the maximum image upload size in megabytes.
func S3MaxUploadSizeMB() int {
	return envInt("S3_MAX_UPLOAD_SIZE_MB", 5)
}

// ResendAPIKey is the API key for the Resend email service.
// Empty means the log fallback sender is used.
func ResendAPIKey() string {
	return env("RESEND_API_KEY", "")
}

// ResendFrom is the sender address for emails sent via Resend.
func ResendFrom() string {
	return env("RESEND_FROM", "MiniMart <onboarding@resend.dev>")
}

// ResendReplyTo is the optional Reply-To address for emails sent via Resend.
func ResendReplyTo() string {
	return env("RESEND_REPLY_TO", "")
}

// ModerationEnabled returns whether AI moderation is enabled.
func ModerationEnabled() bool {
	return envBool("MODERATION_ENABLED", true)
}

// ModerationEndpoint returns the Ollama API endpoint.
func ModerationEndpoint() string {
	return env("MODERATION_ENDPOINT", "http://localhost:11434")
}

// ModerationModel returns the model name to use for moderation.
func ModerationModel() string {
	return env("MODERATION_MODEL", "llama-guard3:1b")
}

// ModerationTimeout returns the timeout for moderation requests.
func ModerationTimeout() time.Duration {
	return envDuration("MODERATION_TIMEOUT", 30*time.Second)
}
