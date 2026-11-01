package outbox

import (
	"context"
	"time"
)

// unlocker releases locks held longer than maxLockTimeDuration, so records held by
// a dispatcher that crashed mid-publish become available again.
type unlocker interface {
	UnlockExpiredMessages(ctx context.Context) error
}

type recordUnlocker struct {
	store               Store
	maxLockTimeDuration time.Duration
	now                 func() time.Time
}

func newRecordUnlocker(store Store, maxLockTimeDuration time.Duration) *recordUnlocker {
	return &recordUnlocker{store: store, maxLockTimeDuration: maxLockTimeDuration, now: time.Now}
}

func (u *recordUnlocker) UnlockExpiredMessages(ctx context.Context) error {
	return u.store.ClearLocksWithDurationBeforeDate(ctx, u.now().UTC().Add(-u.maxLockTimeDuration))
}
