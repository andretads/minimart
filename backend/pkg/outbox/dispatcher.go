package outbox

import (
	"context"
	"crypto/rand"
	"log/slog"
	"sync"
	"time"
)

// RetryPolicy caps how many times a single record is attempted. When disabled,
// failing records are retried forever (bounded only by retention cleanup).
type RetryPolicy struct {
	MaxSendAttemptsEnabled bool
	MaxSendAttempts        int
}

// DispatcherSettings configures the three background workers.
type DispatcherSettings struct {
	// ProcessInterval is how often pending records are polled and published.
	ProcessInterval time.Duration
	// LockCheckerInterval is how often stale locks are reclaimed.
	LockCheckerInterval time.Duration
	// CleanupWorkerInterval is how often delivered records are retired.
	CleanupWorkerInterval time.Duration
	// MaxLockTimeDuration is how long a lock may be held before it is considered
	// abandoned and reclaimed.
	MaxLockTimeDuration time.Duration
	// MessagesRetentionDuration is how long delivered records are kept.
	MessagesRetentionDuration time.Duration
	// RetryPolicy caps delivery attempts per record.
	RetryPolicy RetryPolicy
}

// Dispatcher runs the outbox workers. Each worker polls on its own interval and
// logs failures without stopping: transient broker or database errors are retried
// on the next tick.
type Dispatcher struct {
	recordProcessor processor
	recordUnlocker  unlocker
	recordCleaner   cleaner
	settings        DispatcherSettings
}

// NewDispatcher wires the workers. machineID identifies the host that owns the
// locks (e.g. the hostname). Every dispatcher takes a lock ID of its own, so
// dispatchers that share a host - one per domain inside the worker process -
// never claim and publish each other's records.
func NewDispatcher(store Store, broker MessageBroker, settings DispatcherSettings, machineID string) *Dispatcher {
	lockID := newLockID(machineID)
	return &Dispatcher{
		recordProcessor: newRecordProcessor(store, broker, lockID, settings.RetryPolicy),
		recordUnlocker:  newRecordUnlocker(store, settings.MaxLockTimeDuration),
		recordCleaner:   newRecordCleaner(store, settings.MessagesRetentionDuration),
		settings:        settings,
	}
}

func newLockID(machineID string) string {
	return machineID + "/" + rand.Text()
}

// Run blocks until ctx is cancelled, running the processor, unlocker, and cleaner
// concurrently. It is a no-op-safe background task: each worker runs once
// immediately, then on its ticker.
func (d *Dispatcher) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Go(func() {
		d.runWorker(ctx, "record-processor", d.settings.ProcessInterval, d.recordProcessor.ProcessRecords)
	})
	wg.Go(func() {
		d.runWorker(ctx, "record-unlocker", d.settings.LockCheckerInterval, d.recordUnlocker.UnlockExpiredMessages)
	})
	wg.Go(func() {
		d.runWorker(ctx, "record-cleaner", d.settings.CleanupWorkerInterval, d.recordCleaner.RemoveExpiredMessages)
	})
	wg.Wait()
}

func (d *Dispatcher) runWorker(ctx context.Context, name string, interval time.Duration, work func(context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := work(ctx); err != nil && ctx.Err() == nil {
			slog.Error("outbox worker failed", "worker", name, "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
