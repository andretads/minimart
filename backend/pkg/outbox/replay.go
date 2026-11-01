package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// ReplayOptions bounds one operator replay run. An empty Topic selects every
// abandoned topic, Limit 0 means all, and DryRun only logs the rows it would
// requeue.
type ReplayOptions struct {
	Topic  string
	Limit  int
	DryRun bool
}

// ReplayResult reports how many rows a replay selected and actually requeued.
type ReplayResult struct {
	Selected int
	Requeued int
}

// Replayer requeues abandoned outbox rows so the normal relay publishes them.
// It never publishes directly to Kafka, preserving the outbox's duplicate window
// and per-partition ordering.
type Replayer struct {
	store Store
}

// NewReplayer builds a Replayer over the outbox store.
func NewReplayer(store Store) *Replayer {
	return &Replayer{store: store}
}

// Run lists abandoned rows, optionally filtered by topic, and moves them back
// to pending unless DryRun is set.
func (r *Replayer) Run(ctx context.Context, opts ReplayOptions) (ReplayResult, error) {
	var result ReplayResult
	if opts.Limit == 0 {
		opts.Limit = -1
	}
	if opts.Limit < -1 {
		return result, errors.New("outbox replay: limit cannot be negative")
	}

	var records []AbandonedRecord
	var err error
	if opts.Topic == "" {
		records, err = r.store.ListAbandonedRecords(ctx, opts.Limit)
	} else {
		records, err = r.store.ListAbandonedRecordsByTopic(ctx, opts.Topic, opts.Limit)
	}
	if err != nil {
		return result, fmt.Errorf("list abandoned outbox records: %w", err)
	}

	result.Selected = len(records)
	if opts.DryRun {
		for _, rec := range records {
			slog.InfoContext(ctx, "outbox replay dry run",
				"id", rec.ID,
				"topic", rec.Topic,
				"key", rec.Key,
				"created_on", rec.CreatedOn,
			)
		}
		return result, nil
	}

	ids := make([]string, 0, len(records))
	for _, rec := range records {
		ids = append(ids, rec.ID)
	}
	requeued, err := r.store.RequeueAbandonedRecords(ctx, ids)
	if err != nil {
		return result, fmt.Errorf("requeue abandoned outbox records: %w", err)
	}
	result.Requeued = len(requeued)
	return result, nil
}
