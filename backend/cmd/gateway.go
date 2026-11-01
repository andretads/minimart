package cmd

import (
	"context"
	"errors"
	"log/slog"
	"minimart/conf"
	"minimart/pkg/cors"
	"minimart/pkg/gateway/proxy"
	"minimart/pkg/health"
	"minimart/pkg/logger"
	"minimart/pkg/metrics"
	"minimart/pkg/ratelimit"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

const gatewayShutdownTimeout = 5 * time.Second

func StartGateway() error {
	rec := newMetricsNoPools()

	rdb, err := newRedis()
	if err != nil {
		slog.Warn("gateway redis unavailable; edge rate limiting disabled", "error", err)
		rdb = nil
	}

	upstream := conf.GatewayUpstream()
	p, err := proxy.New(upstream, func(r *http.Request) {
		r.Header.Del("X-Forwarded-Host")
	})
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	health.New(nil).Register(mux)
	mux.Handle("GET /metrics", metrics.Handler())

	corsCfg := cors.Config{
		Origins:       conf.GatewayCORSOrigins(),
		Methods:       conf.GatewayCORSMethods(),
		Headers:       conf.GatewayCORSHeaders(),
		ExposeHeaders: conf.GatewayCORSExposeHeaders(),
		Credentials:   conf.GatewayCORSCredentials(),
		MaxAge:        conf.GatewayCORSMaxAge(),
	}

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/livez" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" {
			mux.ServeHTTP(w, r)
			return
		}
		p.ServeHTTP(w, r)
	})

	handler = cors.Middleware(corsCfg)(handler)

	if rdb != nil {
		rateLimitBreaker := newRedisBreaker("ratelimit")
		rdb.AddHook(metrics.NewRedisHook(rec))
		for _, c := range metrics.RedisPoolCollectors(rdb.Client, conf.RedisPoolSize()) {
			prometheus.MustRegister(c)
		}
		registerBreakerGauge(rateLimitBreaker)
		lim := newRateLimiter(rdb, rateLimitBreaker)
		handler = ratelimit.Middleware(lim, ratelimit.ByHeaderOrIP("Idempotency-Key"), "gateway", rec, handler)
	}

	handler = metrics.HTTPMiddleware(rec, handler)
	handler = logger.HTTPMiddleware(slog.Default(), handler)

	srv := &http.Server{
		Addr:              conf.GatewayHTTPAddr(),
		Handler:           handler,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
		ReadHeaderTimeout: gatewayShutdownTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("gateway starting", "addr", conf.GatewayHTTPAddr(), "upstream", upstream)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("gateway failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("gateway shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), gatewayShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("gateway shutdown failed", "error", err)
	}
	if rdb != nil {
		_ = rdb.Close()
	}

	slog.Info("gateway gracefully stopped")
	return nil
}

func newMetricsNoPools() metrics.Metrics {
	return metrics.NewPrometheusMetrics(map[string]*pgxpool.Pool{})
}
