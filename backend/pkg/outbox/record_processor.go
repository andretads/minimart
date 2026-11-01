package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// unlockTimeout bounds the final lock release so an unresponsive database cannot
// block the dispatcher forever during shutdown.
const unlockTimeout = 5 * time.Second

const defaultBatchSize = 100

// messageIdentity is one Kafka partition: the partition key only orders records
// inside a topic, so different event types for the same aggregate are independent.
type messageIdentity struct {
	topic string
	key   string
}

// processor locks pending records, tries to publish them, and records the outcome.
type processor interface {
	ProcessRecords(ctx context.Context) error
}

type recordProcessor struct {
	store         Store
	broker        MessageBroker
	machineID     string
	batchSize     int
	retrialPolicy RetryPolicy
	relayLock     RelayLock
	now           func() time.Time
}

func newRecordProcessor(store Store, broker MessageBroker, machineID string, batchSize int, retrialPolicy RetryPolicy, relayLock RelayLock) *recordProcessor {
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	if relayLock == nil {
		relayLock = noopRelayLock{}
	}
	return &recordProcessor{
		store:         store,
		broker:        broker,
		machineID:     machineID,
		batchSize:     batchSize,
		retrialPolicy: retrialPolicy,
		relayLock:     relayLock,
		now:           time.Now,
	}
}

// ProcessRecords is the heart of the dispatcher. It locks a bounded batch of
// unprocessed records, publishes them, and always releases its locks on the way
// out.
func (p *recordProcessor) ProcessRecords(ctx context.Context) error {
	acquired, acquireErr := p.relayLock.TryAcquire(ctx)
	if acquireErr != nil {
		return fmt.Errorf("acquire outbox relay lock: %w", acquireErr)
	}
	if !acquired {
		return nil
	}

	releaseCtx, releaseCancel := context.WithTimeout(context.WithoutCancel(ctx), unlockTimeout)
	defer releaseCancel()
	defer func() {
		if err := p.relayLock.Release(releaseCtx); err != nil {
			slog.Warn("outbox relay lock release failed", "error", err)
		}
	}()

	// The deferred unlock must run even if locking fails part-way through, or a
	// transient error would leave records locked until the unlocker reclaims them.
	// It must also survive cancellation: on shutdown the caller's context is
	// already cancelled, and releasing the locks with a cancelled context would
	// leave records locked until a stale-lock reclaimer runs.
	unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), unlockTimeout)
	defer cancel()
	defer func() { _ = p.store.ClearLocksByLockID(unlockCtx, p.machineID) }()

	if err := p.store.ClearAllLocks(ctx); err != nil {
		return fmt.Errorf("clear stale outbox locks: %w", err)
	}

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
	blocked := make(map[messageIdentity]struct{})
	var firstErr error

	for _, rec := range records {
		if err := ctx.Err(); err != nil {
			return errors.Join(firstErr, err)
		}

		identity := messageIdentity{topic: rec.Message.Topic, key: rec.Message.Key}
		if _, ok := blocked[identity]; ok {
			continue
		}

		now := p.now().UTC()
		rec.LastAttemptOn = &now
		rec.NumberOfAttempts++

		if err := p.broker.Send(ctx, rec.Message); err != nil {
			// Keep this row pending for the next poll, but do not skip later rows
			// in the same partition: that would break per-key ordering. Other
			// topics are safe to publish because Kafka does not order them.
			blocked[identity] = struct{}{}
			rec.LockID = nil
			rec.LockedOn = nil
			msg := err.Error()
			rec.Error = &msg
			if p.retrialPolicy.MaxSendAttemptsEnabled && rec.NumberOfAttempts >= p.retrialPolicy.MaxSendAttempts {
				rec.State = MaxAttemptsReached
			}
			if dbErr := p.store.UpdateRecordByID(ctx, rec); dbErr != nil {
				return errors.Join(firstErr, fmt.Errorf("update outbox record %s after send failure: %w", rec.ID, dbErr))
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("send outbox message %s: %w", rec.ID, err)
			}
			continue
		}

		rec.State = Delivered
		rec.LockID = nil
		rec.LockedOn = nil
		rec.ProcessedOn = &now
		rec.Error = nil
		if err := p.store.UpdateRecordByID(ctx, rec); err != nil {
			return errors.Join(firstErr, fmt.Errorf("mark outbox record %s delivered: %w", rec.ID, err))
		}
	}
	return firstErr
}

func (p *recordProcessor) lockUnprocessedEntities(ctx context.Context) error {
	return p.store.UpdateRecordLockByState(ctx, p.machineID, p.now().UTC(), PendingDelivery, p.batchSize)
}
