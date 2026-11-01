package cmd

import (
	"context"
	"errors"
	"log/slog"
	"minimart/conf"
	"minimart/pkg/backpressure"
	"minimart/pkg/health"
	"minimart/pkg/logger"
	"minimart/pkg/metrics"
	"minimart/pkg/module/contracts"
	"minimart/pkg/worker"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	kafkapkg "minimart/pkg/kafka"
)

func StartWorker() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inf, err := newInfra(ctx)
	if err != nil {
		return err
	}
	defer inf.Close(ctx)

	idempotencyBreaker := newRedisBreaker("idempotency")

	gate := backpressure.NewGate()
	consumer := kafkapkg.NewConsumerWithOptions(conf.KafkaBrokers(), inf.producer, inf.rec, kafkapkg.RetryPolicy{
		MaxAttempts: conf.KafkaConsumerMaxAttempts(),
		BaseDelay:   conf.KafkaConsumerRetryBaseDelay(),
		MaxDelay:    conf.KafkaConsumerRetryMaxDelay(),
	}, kafkapkg.ConsumerOptions{Gate: gate, Auth: kafkapkg.Auth{
		Username:  conf.KafkaUsername(),
		Password:  conf.KafkaPassword(),
		Mechanism: conf.KafkaSASLMechanism(),
		EnableTLS: conf.KafkaTLSEnabled(),
	}})
	w := worker.NewWithConfig(consumer, inf.producer, worker.Config{
		RestartMinBackoff: conf.KafkaConsumerRestartMinBackoff(),
		RestartMaxBackoff: conf.KafkaConsumerRestartMaxBackoff(),
		KafkaConcurrency:  conf.KafkaConsumerConcurrency(),
		Gate:              gate,
		Metrics:           inf.rec,
	})

	registry := &contracts.Contracts{}
	modules, _ := newSharedModules(moduleDeps{
		readPool:         inf.readPool,
		writePool:        inf.writePool,
		mongoDB:          inf.mongo.Database(conf.MongoDatabase()),
		registry:         registry,
		rec:              inf.rec,
		producer:         inf.producer,
		rdb:              inf.rdb,
		idem:             newIdempotencyStore(inf.rdb, idempotencyBreaker),
		tokens:           newTokenService(),
		cookieEncryption: newCookieEncryption(),
		cookieConfig:     newCookieConfig(),
	})

	if err := bootModules(ctx, modules, registry); err != nil {
		return err
	}
	for _, m := range modules {
		if err := m.RegisterWorkers(ctx, w); err != nil {
			return err
		}
	}

	if conf.KafkaConsumerPauseEnabled() {
		monitor := backpressure.NewPoolMonitor(gate, map[string]*pgxpool.Pool{"read": inf.readPool, "write": inf.writePool}, conf.KafkaConsumerPausePoolThreshold(), conf.KafkaConsumerPauseInterval(), conf.KafkaConsumerPauseDuration())
		w.AttachBackground(ctx, monitor.Run)
	}

	if conf.KafkaConsumerLagEnabled() {
		lagReader := kafkapkg.NewSaramaLagReader(conf.KafkaBrokers())
		lagMonitor := kafkapkg.NewLagMonitor(lagReader, w.Subscriptions, func(topic, group string, lag int64) {
			inf.rec.SetKafkaConsumerLag(topic, group, lag)
		}, conf.KafkaConsumerLagInterval())
		w.AttachBackground(ctx, lagMonitor.Run)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)

	done := make(chan struct{})
	go func() {
		w.Wait()
		close(done)
	}()

	slog.Info("worker starting; consuming events")

	workerMux := http.NewServeMux()
	workerMux.Handle("GET /metrics", metrics.Handler())
	health.NewWithDependencies(map[string]health.Dependency{
		"read":  {Name: "read", Pinger: inf.readPool, Critical: true},
		"write": {Name: "write", Pinger: inf.writePool, Critical: true},
		"mongo": {Name: "mongo", Pinger: mongoPinger{client: inf.mongo}, Critical: true},
		"redis": {Name: "redis", Pinger: inf.rdb, Critical: conf.RedisRequiredForReadiness()},
	}).Register(workerMux)
	workerSrv := &http.Server{
		Addr:              conf.WorkerHTTPAddr(),
		Handler:           logger.HTTPMiddleware(slog.Default(), workerMux),
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
		ReadHeaderTimeout: shutdownTimeout,
	}
	go func() {
		slog.Info("worker HTTP listening", "addr", conf.WorkerHTTPAddr())
		if err := workerSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("worker HTTP server failed", "error", err)
		}
	}()

	<-stop
	slog.Info("shutdown signal received")

	cancel()

	select {
	case <-done:
	case <-time.After(shutdownTimeout):
		slog.Warn("worker shutdown timed out")
	}

	metricsCtx, metricsCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer metricsCancel()
	if err := workerSrv.Shutdown(metricsCtx); err != nil {
		slog.Error("worker HTTP server shutdown failed", "error", err)
	}

	slog.Info("worker gracefully stopped")
	return nil
}
