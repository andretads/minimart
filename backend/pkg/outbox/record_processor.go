package outbox

import (
	"context"
	"fmt"
	"time"
)

// unlockTimeout bounds the final lock release so an unresponsive database cannot
// block the dispatcher forever during shutdown.
const unlockTimeout = 5 * time.Second

// processor locks pending records, tries to publish them, and records the outcome.
type processor interface {
	ProcessRecords(ctx context.Context) error
}

type recordProcessor struct {
	store         Store
	broker        MessageBroker
	machineID     string
	retrialPolicy RetryPolicy
	now           func() time.Time
}

func newRecordProcessor(store Store, broker MessageBroker, machineID string, retrialPolicy RetryPolicy) *recordProcessor {
	return &recordProcessor{
		store:         store,
		broker:        broker,
		machineID:     machineID,
		retrialPolicy: retrialPolicy,
		now:           time.Now,
	}
}

// ProcessRecords is the heart of the dispatcher. It locks the unprocessed records
// so no other dispatcher picks them up, publishes them, and always releases its
// locks on the way out.
func (p *recordProcessor) ProcessRecords(ctx context.Context) error {
	// The deferred unlock must run even if locking fails part-way through, or a
	// transient error would leave records locked until the unlocker reclaims them.
	// It must also survive cancellation: on shutdown the caller's context is
	// already cancelled, and releasing the locks with a cancelled context would
	// leave records locked until a stale-lock reclaimer runs.
	unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), unlockTimeout)
	defer cancel()
	defer func() { _ = p.store.ClearLocksByLockID(unlockCtx, p.machineID) }()

	if err := p.lockUnprocessedEntities(ctx); err != nil {
		return err
	}

	records, err := p.store.GetRecordsByLockID(ctx, p.machineID)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	return p.publishMessages(ctx, records)
}

func (p *recordProcessor) publishMessages(ctx context.Context, records []Record) error {
	for i := range records {
		rec := records[i]
		now := p.now().UTC()
		rec.LastAttemptOn = &now
		rec.NumberOfAttempts++

		if err := p.broker.Send(ctx, rec.Message); err != nil {
			rec.LockID = nil
			rec.LockedOn = nil
			msg := err.Error()
			rec.Error = &msg
			// Stop at the first failure: preserving per-key order means not
			// skipping ahead of an undelivered message.
			if p.retrialPolicy.MaxSendAttemptsEnabled && rec.NumberOfAttempts >= p.retrialPolicy.MaxSendAttempts {
				rec.State = MaxAttemptsReached
			}
			if dbErr := p.store.UpdateRecordByID(ctx, rec); dbErr != nil {
				return fmt.Errorf("update outbox record %s after send failure: %w", rec.ID, dbErr)
			}
			return fmt.Errorf("send outbox message %s: %w", rec.ID, err)
		}

		rec.State = Delivered
		rec.LockID = nil
		rec.LockedOn = nil
		rec.ProcessedOn = &now
		rec.Error = nil
		if err := p.store.UpdateRecordByID(ctx, rec); err != nil {
			return fmt.Errorf("mark outbox record %s delivered: %w", rec.ID, err)
		}
	}
	return nil
}

func (p *recordProcessor) lockUnprocessedEntities(ctx context.Context) error {
	return p.store.UpdateRecordLockByState(ctx, p.machineID, p.now().UTC(), PendingDelivery)
}
