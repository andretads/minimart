package outbox

import (
	"context"
	"log/slog"
	"time"
)

const defaultStatsInterval = 30 * time.Second

// Gauge state label values. They describe the four buckets exported by the
// outbox stats collector.
const (
	StateLabelPending   = "pending"
	StateLabelLocked    = "locked"
	StateLabelDelivered = "delivered"
	StateLabelAbandoned = "abandoned"
)

// StatsRecorder is the observability port used by MetricsCollector. metrics.Metrics
// implements it, while tests can inject a recorder that only captures values.
type StatsRecorder interface {
	SetOutboxRecords(state string, count int64)
	SetOutboxOldestRecordAge(state string, age time.Duration)
}

// MetricsCollector samples the outbox table and exports backlog gauges. One
// collector is attached to the worker role; per-domain relays only publish rows.
type MetricsCollector struct {
	store    Store
	recorder StatsRecorder
	interval time.Duration
}

// NewMetricsCollector builds a collector that samples every interval. A
// non-positive interval uses the default.
func NewMetricsCollector(store Store, recorder StatsRecorder, interval time.Duration) *MetricsCollector {
	if interval <= 0 {
		interval = defaultStatsInterval
	}
	return &MetricsCollector{store: store, recorder: recorder, interval: interval}
}

// Run samples immediately, then on each tick, until ctx is cancelled.
func (c *MetricsCollector) Run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		if err := c.collect(ctx); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "outbox stats collection failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *MetricsCollector) collect(ctx context.Context) error {
	stats, err := c.store.GetStats(ctx)
	if err != nil {
		return err
	}

	c.recorder.SetOutboxRecords(StateLabelPending, stats.Pending)
	c.recorder.SetOutboxRecords(StateLabelLocked, stats.Locked)
	c.recorder.SetOutboxRecords(StateLabelDelivered, stats.Delivered)
	c.recorder.SetOutboxRecords(StateLabelAbandoned, stats.Abandoned)

	c.recorder.SetOutboxOldestRecordAge(StateLabelPending, stats.OldestPending)
	c.recorder.SetOutboxOldestRecordAge(StateLabelLocked, stats.OldestLocked)
	c.recorder.SetOutboxOldestRecordAge(StateLabelDelivered, stats.OldestDelivered)
	c.recorder.SetOutboxOldestRecordAge(StateLabelAbandoned, stats.OldestAbandoned)
	return nil
}
