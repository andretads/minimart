package outbox

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Stats is a point-in-time view of the outbox table used for backlog gauges.
type Stats struct {
	Pending         int64
	Locked          int64
	Delivered       int64
	Abandoned       int64
	OldestPending   time.Duration
	OldestLocked    time.Duration
	OldestDelivered time.Duration
	OldestAbandoned time.Duration
}

// AbandonedRecord is the replayable summary of a MaxAttemptsReached row.
type AbandonedRecord struct {
	ID        string
	Topic     string
	Key       string
	CreatedOn time.Time
}

// Store is the persistence port the Publisher and Dispatcher depend on. A SQL
// driver package implements it; pkg/outbox/postgres is the Postgres adapter.
type Store interface {
	// AddRecordTx stores a record inside the caller's transaction, so the outbox
	// row commits atomically with the aggregate it describes.
	AddRecordTx(ctx context.Context, record Record, tx pgx.Tx) error
	// GetRecordsByLockID returns the records currently locked by lockID, oldest first.
	GetRecordsByLockID(ctx context.Context, lockID string) ([]Record, error)
	// UpdateRecordLockByState claims at most limit unlocked records in the given
	// state for lockID.
	UpdateRecordLockByState(ctx context.Context, lockID string, lockedOn time.Time, state RecordState, limit int) error
	// UpdateRecordByID persists a record's delivery outcome.
	UpdateRecordByID(ctx context.Context, record Record) error
	// ClearLocksWithDurationBeforeDate releases locks older than the given time,
	// reclaiming records abandoned by a crashed dispatcher.
	ClearLocksWithDurationBeforeDate(ctx context.Context, before time.Time) error
	// ClearLocksByLockID releases every lock held by lockID.
	ClearLocksByLockID(ctx context.Context, lockID string) error
	// ClearAllLocks releases every row lock still held after a relay takeover.
	ClearAllLocks(ctx context.Context) error
	// RemoveDeliveredRecordsBeforeDatetime deletes delivered rows created before
	// expiryTime. Undelivered and abandoned rows are retained for replay.
	RemoveDeliveredRecordsBeforeDatetime(ctx context.Context, expiryTime time.Time) error
	// GetStats returns current counts and oldest-record ages by state.
	GetStats(ctx context.Context) (Stats, error)
	// ListAbandonedRecords returns up to limit abandoned rows, oldest first; a
	// negative limit means all rows.
	ListAbandonedRecords(ctx context.Context, limit int) ([]AbandonedRecord, error)
	// ListAbandonedRecordsByTopic is ListAbandonedRecords filtered by topic.
	ListAbandonedRecordsByTopic(ctx context.Context, topic string, limit int) ([]AbandonedRecord, error)
	// RequeueAbandonedRecords moves the named abandoned rows back to pending and
	// returns the IDs it actually updated.
	RequeueAbandonedRecords(ctx context.Context, ids []string) ([]string, error)
}
