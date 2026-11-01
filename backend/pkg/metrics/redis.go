package metrics

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	goredis "github.com/redis/go-redis/v9"
)

// redisHook records per-command Redis metrics and tracks availability.
type redisHook struct {
	rec Metrics
	up  atomic.Bool
}

// NewRedisHook returns a go-redis hook that records command counters/latency and
// an availability flag. Call AddHook on the client to install it.
func NewRedisHook(rec Metrics) *redisHook {
	h := &redisHook{rec: rec}
	h.up.Store(true)
	return h
}

// Up reports whether Redis is currently reachable.
func (h *redisHook) Up() bool { return h.up.Load() }

func (h *redisHook) DialHook(next goredis.DialHook) goredis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		c, e := next(ctx, network, addr)
		if e != nil {
			h.up.Store(false)
		} else {
			h.up.Store(true)
		}
		return c, e
	}
}

func (h *redisHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		dur := time.Since(start)
		status := "ok"
		if err != nil {
			status = "error"
			h.up.Store(false)
		} else {
			h.up.Store(true)
		}
		h.rec.IncRedisCommand(cmd.Name(), status)
		h.rec.ObserveRedisCommandLatency(cmd.Name(), dur)
		return err
	}
}

func (h *redisHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []goredis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)
		dur := time.Since(start)
		status := "ok"
		if err != nil {
			status = "error"
			h.up.Store(false)
		} else {
			h.up.Store(true)
		}
		for _, cmd := range cmds {
			h.rec.IncRedisCommand(cmd.Name(), status)
		}
		// Attribute the whole pipeline's latency to a synthetic command label so
		// per-command histograms are not inflated by batch cost.
		h.rec.ObserveRedisCommandLatency("pipeline", dur)
		return err
	}
}

// RedisPoolCollectors exposes go-redis pool statistics as scrape-time gauges.
func RedisPoolCollectors(client *goredis.Client, maxConns int) []prometheus.Collector {
	labels := prometheus.Labels{}
	_ = labels
	opts := func(name, help string) prometheus.GaugeOpts {
		return prometheus.GaugeOpts{Namespace: namespace, Subsystem: "redis_pool", Name: name, Help: help}
	}
	coll := []prometheus.Collector{
		prometheus.NewGaugeFunc(opts("total_connections", "Number of connections currently in the pool."), func() float64 {
			return float64(client.PoolStats().TotalConns)
		}),
		prometheus.NewGaugeFunc(opts("idle_connections", "Number of idle connections in the pool."), func() float64 {
			return float64(client.PoolStats().IdleConns)
		}),
		prometheus.NewGaugeFunc(opts("stale_connections", "Number of stale connections removed from the pool."), func() float64 {
			return float64(client.PoolStats().StaleConns)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{Namespace: namespace, Subsystem: "redis_pool", Name: "max_connections", Help: "Configured maximum pool size."}, func() float64 {
			return float64(maxConns)
		}),
	}
	// Counter-style cumulative stats are exposed as GaugeFuncs reading the
	// monotonically increasing pool counters; Prometheus handles rate() fine.
	counterOpts := func(name, help string) prometheus.GaugeOpts {
		return prometheus.GaugeOpts{Namespace: namespace, Subsystem: "redis_pool", Name: name, Help: help}
	}
	coll = append(coll,
		prometheus.NewGaugeFunc(counterOpts("hits_total", "Cumulative pool hits."), func() float64 { return float64(client.PoolStats().Hits) }),
		prometheus.NewGaugeFunc(counterOpts("misses_total", "Cumulative pool misses."), func() float64 { return float64(client.PoolStats().Misses) }),
		prometheus.NewGaugeFunc(counterOpts("timeouts_total", "Cumulative pool timeouts."), func() float64 { return float64(client.PoolStats().Timeouts) }),
	)
	return coll
}
