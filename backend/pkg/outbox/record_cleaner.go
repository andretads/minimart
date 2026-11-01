package outbox

import (
	"context"
	"time"
)

// cleaner deletes delivered records once they are older than the retention
// duration. Undelivered and abandoned rows are kept for retry or replay.
type cleaner interface {
	RemoveExpiredMessages(ctx context.Context) error
}

type recordCleaner struct {
	store             Store
	maxRecordLifetime time.Duration
	now               func() time.Time
}

func newRecordCleaner(store Store, maxRecordLifetime time.Duration) *recordCleaner {
	return &recordCleaner{store: store, maxRecordLifetime: maxRecordLifetime, now: time.Now}
}

func (c *recordCleaner) RemoveExpiredMessages(ctx context.Context) error {
	return c.store.RemoveDeliveredRecordsBeforeDatetime(ctx, c.now().UTC().Add(-c.maxRecordLifetime))
}
