// Package redis builds the shared Redis client MiniMart uses for the payment
// domain's idempotency-key cache and the global HTTP rate limiter. It is a thin
// wrapper over go-redis so the rest of the system depends on this constructor -
// and on health.Pinger, which Client satisfies - rather than on the client
// library directly.
//
// Redis is an accelerator, never the system of record: the payments table's
// unique index on idempotency_key remains authoritative, so Redis being
// unavailable degrades to database-only idempotency instead of allowing
// duplicates.
package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Options configures the Redis client. Zero values are replaced with sensible
// bounded defaults by NewWithOptions.
type Options struct {
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration
	MaxRetries   int
	PoolSize     int
}

// Client wraps a go-redis client. The embedded client exposes the full command
// surface to the idempotency and rate-limit adapters; the Ping method below
// satisfies health.Pinger so /readyz covers Redis too.
type Client struct {
	*goredis.Client
}

// RedisClientAdapter adapts *Client to authz.RedisClient.
type RedisClientAdapter struct {
	*Client
}

func (a *RedisClientAdapter) Get(ctx context.Context, key string) *goredis.StringCmd {
	return a.Client.Get(ctx, key)
}

func (a *RedisClientAdapter) Set(ctx context.Context, key string, value any, ttl time.Duration) *goredis.StatusCmd {
	return a.Client.Set(ctx, key, value, ttl)
}

// New parses url (redis://[:password@]host:port/db) and returns a client with
// bounded default timeouts. It does not connect eagerly - call Ping to verify
// reachability at boot.
func New(url string) (*Client, error) {
	return NewWithOptions(url, Options{})
}

// NewWithOptions parses url and applies the supplied (or default) timeouts and
// retry policy. A MaxRetries value of zero disables retries.
func NewWithOptions(url string, opts Options) (*Client, error) {
	parsed, err := goredis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	if opts.DialTimeout > 0 {
		parsed.DialTimeout = opts.DialTimeout
	} else if parsed.DialTimeout == 0 {
		parsed.DialTimeout = 2 * time.Second
	}
	if opts.ReadTimeout > 0 {
		parsed.ReadTimeout = opts.ReadTimeout
	} else if parsed.ReadTimeout == 0 {
		parsed.ReadTimeout = 200 * time.Millisecond
	}
	if opts.WriteTimeout > 0 {
		parsed.WriteTimeout = opts.WriteTimeout
	} else if parsed.WriteTimeout == 0 {
		parsed.WriteTimeout = 200 * time.Millisecond
	}
	if opts.PoolTimeout > 0 {
		parsed.PoolTimeout = opts.PoolTimeout
	} else if parsed.PoolTimeout == 0 {
		parsed.PoolTimeout = 200 * time.Millisecond
	}
	parsed.MaxRetries = opts.MaxRetries
	if opts.PoolSize > 0 {
		parsed.PoolSize = opts.PoolSize
	}
	parsed.ContextTimeoutEnabled = true
	return &Client{Client: goredis.NewClient(parsed)}, nil
}

// Ping verifies Redis is reachable. It matches health.Pinger (context in, error
// out) rather than the library's *StatusCmd-returning Ping.
func (c *Client) Ping(ctx context.Context) error {
	return c.Client.Ping(ctx).Err()
}
