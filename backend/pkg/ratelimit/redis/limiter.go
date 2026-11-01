// Package redis is the Redis adapter of ratelimit.Limiter.
package redis

import (
	"context"
	"fmt"
	"minimart/pkg/ratelimit"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const incrScript = `
local t = redis.call('TIME')
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)
local capacity = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])

local d = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(d[1])
local last = tonumber(d[2])
if tokens == nil then
  tokens = capacity
  last = now
end

tokens = math.min(capacity, tokens + math.max(0, now - last) * rate)

local allowed = 0
local retry_ms = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
else
  retry_ms = math.ceil((1 - tokens) / rate)
end

redis.call('HSET', KEYS[1], 'tokens', tostring(tokens), 'ts', tostring(now))
local expire_ms = math.max(1, math.ceil((capacity - tokens) / rate))
redis.call('PEXPIRE', KEYS[1], expire_ms)
return {allowed, math.floor(tokens), retry_ms}
`

type Limiter struct {
	rdb    *goredis.Client
	script *goredis.Script
	burst  int
	rate   float64
}

var _ ratelimit.Limiter = (*Limiter)(nil)

func New(rdb *goredis.Client, burst, refillTokens int, refillInterval time.Duration) *Limiter {
	if burst < 1 {
		burst = 1
	}
	if refillTokens < 1 {
		refillTokens = 1
	}
	if refillInterval.Milliseconds() < 1 {
		refillInterval = time.Millisecond
	}
	rate := float64(refillTokens) / float64(refillInterval.Milliseconds())
	return &Limiter{rdb: rdb, script: goredis.NewScript(incrScript), burst: burst, rate: rate}
}

// Namespace isolates rate-limit keys from other Redis adapters.
const ratelimitPrefix = "ratelimit:"

func (l *Limiter) Allow(ctx context.Context, key string) (ratelimit.Result, error) {
	nsKey := ratelimitPrefix + key
	raw, err := l.script.Run(ctx, l.rdb, []string{nsKey}, l.burst, l.rate).Result()
	if err != nil {
		return ratelimit.Result{}, err
	}
	vals, ok := raw.([]any)
	if !ok || len(vals) != 3 {
		return ratelimit.Result{}, fmt.Errorf("unexpected rate limit script result %T", raw)
	}
	allowed, ok := vals[0].(int64)
	if !ok {
		return ratelimit.Result{}, fmt.Errorf("unexpected rate limit allowed type %T", vals[0])
	}
	remaining, ok := vals[1].(int64)
	if !ok {
		return ratelimit.Result{}, fmt.Errorf("unexpected rate limit remaining type %T", vals[1])
	}
	retryMillis, ok := vals[2].(int64)
	if !ok {
		return ratelimit.Result{}, fmt.Errorf("unexpected rate limit retry type %T", vals[2])
	}
	res := ratelimit.Result{
		Allowed:    allowed == 1,
		Remaining:  int(remaining),
		RetryAfter: time.Duration(retryMillis) * time.Millisecond,
	}
	return res, nil
}
