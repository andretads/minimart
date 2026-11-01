package cmd

import (
	"context"
	"errors"
	"log/slog"
	"minimart/conf"
	"minimart/docs"
	"minimart/internal/avatar"
	"minimart/internal/cart"
	"minimart/internal/showcase"
	"minimart/internal/wishlist"
	authpkg "minimart/pkg/auth"
	"minimart/pkg/authz"
	"minimart/pkg/health"
	"minimart/pkg/logger"
	"minimart/pkg/metrics"
	"minimart/pkg/module/contracts"
	"minimart/pkg/ratelimit"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

const shutdownTimeout = 5 * time.Second

func StartServer() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inf, err := newInfra(ctx)
	if err != nil {
		return err
	}
	defer inf.Close(ctx)

	idempotencyBreaker := newRedisBreaker("idempotency")
	rateLimitBreaker := newRedisBreaker("ratelimit")
	inf.registerPoolGauges()
	registerBreakerGauge(idempotencyBreaker)
	registerBreakerGauge(rateLimitBreaker)

	mux := http.NewServeMux()
	health.NewWithDependencies(map[string]health.Dependency{
		"read":  {Name: "read", Pinger: inf.readPool, Critical: true},
		"write": {Name: "write", Pinger: inf.writePool, Critical: true},
		"mongo": {Name: "mongo", Pinger: mongoPinger{client: inf.mongo}, Critical: true},
		"redis": {Name: "redis", Pinger: inf.rdb, Critical: conf.RedisRequiredForReadiness()},
	}).Register(mux)
	mux.Handle("GET /metrics", metrics.Handler())

	if conf.Env() != "production" {
		docs.SwaggerInfo.Version = version
		mux.Handle("/swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))
	}

	registry := &contracts.Contracts{}
	tokens := newTokenService()
	cookieEncryption := newCookieEncryption()
	cookieConfig := newCookieConfig()
	modules, authModule := newSharedModules(moduleDeps{
		readPool:         inf.readPool,
		writePool:        inf.writePool,
		mongoDB:          inf.mongo.Database(conf.MongoDatabase()),
		registry:         registry,
		rec:              inf.rec,
		producer:         inf.producer,
		rdb:              inf.rdb,
		idem:             newIdempotencyStore(inf.rdb, idempotencyBreaker),
		tokens:           tokens,
		cookieEncryption: cookieEncryption,
		cookieConfig:     cookieConfig,
	})
	modules = append(modules,
		avatar.NewModule(inf.readPool, inf.writePool, inf.rec),
		cart.NewModule(inf.mongo.Database(conf.MongoDatabase()), registry, inf.rec),
		wishlist.NewModule(inf.mongo.Database(conf.MongoDatabase()), registry, inf.rec),
		showcase.NewModule(inf.readPool),
	)

	if err := bootModules(ctx, modules, registry); err != nil {
		return err
	}

	if err := authModule.EnsureAdmins(ctx, conf.AdminUsernames()); err != nil {
		return err
	}

	authzStore := newAuthzStore(authModule.Service(), inf.rdb)

	for _, m := range modules {
		if err := m.RegisterHTTP(ctx, mux); err != nil {
			return err
		}
	}

	publicPaths := authpkg.PublicPaths(authpkg.PublicRoutePaths...)

	optionalPaths := authpkg.PublicPaths("/v1/auth/me")
	var handler http.Handler = mux
	handler = ratelimit.Middleware(newRateLimiter(inf.rdb, rateLimitBreaker), ratelimit.ByHeaderOrIP("Idempotency-Key"), "global", inf.rec, handler)

	handler = authz.Middleware(authzStore, inf.rec, mux, handler)
	handler = authpkg.Middleware(tokens, cookieEncryption, cookieConfig, publicPaths, optionalPaths, mux, handler)
	handler = metrics.HTTPMiddleware(inf.rec, handler)
	handler = logger.HTTPMiddleware(slog.Default(), handler)

	srv := &http.Server{
		Addr:              conf.ServerHTTPAddr(),
		Handler:           handler,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
		ReadHeaderTimeout: shutdownTimeout,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)

	go func() {
		slog.Info("server starting", "addr", conf.ServerHTTPAddr())

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown failed", "error", err)
	}

	cancel()
	slog.Info("server gracefully stopped")
	return nil
}
