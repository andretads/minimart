package outbox

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Store is the persistence port the Publisher and Dispatcher depend on. A SQL
// driver package implements it; pkg/outbox/postgres is the Postgres adapter.
type Store interface {
	// AddRecordTx stores a record inside the caller's transaction, so the outbox
	// row commits atomically with the aggregate it describes.
	AddRecordTx(ctx context.Context, record Record, tx pgx.Tx) error
	// GetRecordsByLockID returns the records currently locked by lockID.
	GetRecordsByLockID(ctx context.Context, lockID string) ([]Record, error)
	// UpdateRecordLockByState locks every unlocked record in the given state.
	UpdateRecordLockByState(ctx context.Context, lockID string, lockedOn time.Time, state RecordState) error
	// UpdateRecordByID persists a record's delivery outcome.
	UpdateRecordByID(ctx context.Context, record Record) error
	// ClearLocksWithDurationBeforeDate releases locks older than the given time,
	// reclaiming records abandoned by a crashed dispatcher.
	ClearLocksWithDurationBeforeDate(ctx context.Context, before time.Time) error
	// ClearLocksByLockID releases every lock held by lockID.
	ClearLocksByLockID(ctx context.Context, lockID string) error
	// RemoveRecordsBeforeDatetime deletes records created before expiryTime that
	// are no longer pending, so retention never drops an undelivered message.
	RemoveRecordsBeforeDatetime(ctx context.Context, expiryTime time.Time) error
}
