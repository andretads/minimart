// Package worker runs a domain's Kafka consumers. It is the worker-process
// equivalent of an http.ServeMux: modules register handlers onto it during the
// RegisterWorkers boot phase, and each handler runs as one consumer-group loop.
package worker

import (
	"context"
	"log/slog"
	"minimart/pkg/backpressure"
	"minimart/pkg/kafka"
	"minimart/pkg/metrics"
	"sort"
	"sync"
	"time"
)

// Subscriber is the consumer port the worker needs. kafka.Consumer implements it;
// defining it here (on the consumer side) keeps the worker testable with a fake.
type Subscriber interface {
	Subscribe(ctx context.Context, topic kafka.Topic, group string, handler func(ctx context.Context, msg []byte) error) error
}

// Compile-time proof the production consumer satisfies the worker's port.
var _ Subscriber = (*kafka.Consumer)(nil)

// Config controls consumer supervision. Zero values fall back to defaults.
type Config struct {
	// RestartMinBackoff is the first wait after a consumer stops unexpectedly.
	RestartMinBackoff time.Duration
	// RestartMaxBackoff caps the restart wait.
	RestartMaxBackoff time.Duration
	// KafkaConcurrency is the number of supervised consumer replicas run per
	// (topic, group) subscription. Zero falls back to one.
	KafkaConcurrency int
	// Gate pauses consumption when dependencies are saturated. May be nil.
	Gate *backpressure.Gate
	// Metrics records restarts; nil uses a no-op recorder.
	Metrics metrics.Metrics
}

// Options controls a single AttachKafka subscription.
type Options struct {
	// Concurrency overrides Config.KafkaConcurrency for this subscription.
	// Zero falls back to the worker default.
	Concurrency int
}

type Worker struct {
	consumer    Subscriber
	producer    kafka.Publisher
	m           metrics.Metrics
	restartMin  time.Duration
	restartMax  time.Duration
	concurrency int
	gate        *backpressure.Gate
	sleep       func(context.Context, time.Duration) error
	wg          sync.WaitGroup

	mu            sync.Mutex
	subscriptions map[kafka.Subscription]struct{}
}

func New(consumer Subscriber, producer kafka.Publisher) *Worker {
	return NewWithConfig(consumer, producer, Config{})
}

// NewWithConfig builds a worker with explicit supervision settings.
func NewWithConfig(consumer Subscriber, producer kafka.Publisher, cfg Config) *Worker {
	if cfg.RestartMinBackoff <= 0 {
		cfg.RestartMinBackoff = time.Second
	}
	if cfg.RestartMaxBackoff < cfg.RestartMinBackoff {
		cfg.RestartMaxBackoff = 30 * time.Second
	}
	if cfg.KafkaConcurrency <= 0 {
		cfg.KafkaConcurrency = 1
	}
	if cfg.Metrics == nil {
		cfg.Metrics = metrics.Noop{}
	}
	return &Worker{
		consumer:      consumer,
		producer:      producer,
		m:             cfg.Metrics,
		restartMin:    cfg.RestartMinBackoff,
		restartMax:    cfg.RestartMaxBackoff,
		concurrency:   cfg.KafkaConcurrency,
		gate:          cfg.Gate,
		sleep:         sleepContext,
		subscriptions: make(map[kafka.Subscription]struct{}),
	}
}

// Producer exposes the shared publisher to modules that also emit events.
func (w *Worker) Producer() kafka.Publisher { return w.producer }

// Gate exposes the pause gate, if configured.
func (w *Worker) Gate() *backpressure.Gate { return w.gate }

// Subscriptions returns the unique (topic, group) pairs registered so far.
func (w *Worker) Subscriptions() []kafka.Subscription {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]kafka.Subscription, 0, len(w.subscriptions))
	for sub := range w.subscriptions {
		out = append(out, sub)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Topic != out[j].Topic {
			return out[i].Topic < out[j].Topic
		}
		return out[i].Group < out[j].Group
	})
	return out
}

// AttachKafka registers a handler for a topic under a consumer group. It starts
// as many supervised consumer replicas as the configured concurrency, each in
// its own goroutine until ctx is cancelled. A consumer that stops unexpectedly
// is restarted with exponential backoff.
func (w *Worker) AttachKafka(ctx context.Context, topic kafka.Topic, group string, handler func(ctx context.Context, msg []byte) error) {
	w.AttachKafkaWithOptions(ctx, topic, group, handler, Options{})
}

// AttachKafkaWithOptions is like AttachKafka but allows per-subscription overrides.
func (w *Worker) AttachKafkaWithOptions(ctx context.Context, topic kafka.Topic, group string, handler func(ctx context.Context, msg []byte) error, opts Options) {
	concurrency := w.concurrency
	if opts.Concurrency > 0 {
		concurrency = opts.Concurrency
	}
	if concurrency < 1 {
		concurrency = 1
	}

	w.mu.Lock()
	w.subscriptions[kafka.Subscription{Topic: topic, Group: group}] = struct{}{}
	w.mu.Unlock()

	for replica := range concurrency {
		w.wg.Go(func() {
			backoff := w.restartMin
			for {
				err := w.consumer.Subscribe(ctx, topic, group, handler)
				if ctx.Err() != nil {
					return
				}
				if err != nil {
					slog.Error("consumer stopped", "topic", topic.String(), "group", group, "replica", replica, "error", err)
				} else {
					slog.Warn("consumer stopped unexpectedly", "topic", topic.String(), "group", group, "replica", replica)
				}
				w.m.IncKafkaConsumerRestart(topic.String(), group)

				if err := w.sleep(ctx, backoff); err != nil {
					return
				}
				backoff = min(backoff*2, w.restartMax)
			}
		})
	}
}

// AttachBackground registers a long-running background loop (such as the outbox
// dispatcher) that is not a Kafka consumer. It runs in its own goroutine for the
// worker's lifetime and participates in graceful shutdown: run must return when
// ctx is cancelled.
func (w *Worker) AttachBackground(ctx context.Context, run func(ctx context.Context)) {
	w.wg.Go(func() { run(ctx) })
}

// Wait blocks until every consumer goroutine has exited.
func (w *Worker) Wait() { w.wg.Wait() }

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
