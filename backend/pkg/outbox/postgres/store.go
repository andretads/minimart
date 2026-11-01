package postgres

import (
	"context"
	"minimart/pkg/outbox"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the Postgres adapter of outbox.Store. Its generated queries come from
// queries.sql; this file is the hand-written translation between the sqlc row
// types and the outbox domain types.
type Store struct {
	q *Queries
}

var _ outbox.Store = (*Store)(nil)

// NewStore builds a Store over the write pool. The dispatcher both reads and
// writes records, so it uses the primary - never the replica.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{q: New(pool)}
}

func (s *Store) AddRecordTx(ctx context.Context, record outbox.Record, tx pgx.Tx) error {
	return s.q.WithTx(tx).AddRecord(ctx, AddRecordParams{
		ID:         record.ID,
		Topic:      record.Message.Topic,
		MessageKey: record.Message.Key,
		Body:       record.Message.Body,
		State:      int32(record.State), //nolint:gosec // RecordState is a bounded enum (0-2)
		CreatedOn:  record.CreatedOn,
	})
}

func (s *Store) GetRecordsByLockID(ctx context.Context, lockID string) ([]outbox.Record, error) {
	rows, err := s.q.GetRecordsByLockID(ctx, &lockID)
	if err != nil {
		return nil, err
	}
	records := make([]outbox.Record, 0, len(rows))
	for _, r := range rows {
		records = append(records, toRecord(r))
	}
	return records, nil
}

func (s *Store) UpdateRecordLockByState(ctx context.Context, lockID string, lockedOn time.Time, state outbox.RecordState, limit int) error {
	return s.q.LockRecordsByState(ctx, LockRecordsByStateParams{
		State:    int32(state), //nolint:gosec // RecordState is a bounded enum (0-2)
		Limit:    int32(limit), //nolint:gosec // operator-configured batch size is small
		LockedBy: &lockID,
		LockedOn: &lockedOn,
	})
}

func (s *Store) UpdateRecordByID(ctx context.Context, record outbox.Record) error {
	return s.q.UpdateRecord(ctx, UpdateRecordParams{
		ID:               record.ID,
		State:            int32(record.State), //nolint:gosec // RecordState is a bounded enum (0-2)
		ProcessedOn:      record.ProcessedOn,
		LockedBy:         record.LockID,
		LockedOn:         record.LockedOn,
		NumberOfAttempts: int32(record.NumberOfAttempts), //nolint:gosec // capped by the retry policy; far below MaxInt32
		LastAttemptedOn:  record.LastAttemptOn,
		Error:            record.Error,
	})
}

func (s *Store) ClearLocksWithDurationBeforeDate(ctx context.Context, before time.Time) error {
	return s.q.ClearStaleLocks(ctx, &before)
}

func (s *Store) ClearLocksByLockID(ctx context.Context, lockID string) error {
	return s.q.ClearLocksByLockID(ctx, &lockID)
}

func (s *Store) ClearAllLocks(ctx context.Context) error {
	return s.q.ClearAllLocks(ctx)
}

func (s *Store) RemoveDeliveredRecordsBeforeDatetime(ctx context.Context, expiryTime time.Time) error {
	return s.q.DeleteDeliveredRecordsBefore(ctx, DeleteDeliveredRecordsBeforeParams{
		State:     int32(outbox.Delivered),
		CreatedOn: expiryTime,
	})
}

func (s *Store) GetStats(ctx context.Context) (outbox.Stats, error) {
	row, err := s.q.GetOutboxStats(ctx)
	if err != nil {
		return outbox.Stats{}, err
	}
	return outbox.Stats{
		Pending:         row.Pending,
		Locked:          row.Locked,
		Delivered:       row.Delivered,
		Abandoned:       row.Abandoned,
		OldestPending:   secondsToDuration(row.OldestPendingSeconds),
		OldestLocked:    secondsToDuration(row.OldestLockedSeconds),
		OldestDelivered: secondsToDuration(row.OldestDeliveredSeconds),
		OldestAbandoned: secondsToDuration(row.OldestAbandonedSeconds),
	}, nil
}

func (s *Store) ListAbandonedRecords(ctx context.Context, limit int) ([]outbox.AbandonedRecord, error) {
	rows, err := s.q.ListAbandonedRecords(ctx, ListAbandonedRecordsParams{
		State: int32(outbox.MaxAttemptsReached),
		Limit: int32(limit), //nolint:gosec // operator-configured batch size is small
	})
	if err != nil {
		return nil, err
	}
	records := make([]outbox.AbandonedRecord, 0, len(rows))
	for _, r := range rows {
		records = append(records, outbox.AbandonedRecord{
			ID:        r.ID,
			Topic:     r.Topic,
			Key:       r.MessageKey,
			CreatedOn: r.CreatedOn,
		})
	}
	return records, nil
}

func (s *Store) ListAbandonedRecordsByTopic(ctx context.Context, topic string, limit int) ([]outbox.AbandonedRecord, error) {
	rows, err := s.q.ListAbandonedRecordsByTopic(ctx, ListAbandonedRecordsByTopicParams{
		State: int32(outbox.MaxAttemptsReached),
		Topic: topic,
		Limit: int32(limit), //nolint:gosec // operator-configured batch size is small
	})
	if err != nil {
		return nil, err
	}
	records := make([]outbox.AbandonedRecord, 0, len(rows))
	for _, r := range rows {
		records = append(records, outbox.AbandonedRecord{
			ID:        r.ID,
			Topic:     r.Topic,
			Key:       r.MessageKey,
			CreatedOn: r.CreatedOn,
		})
	}
	return records, nil
}

func (s *Store) RequeueAbandonedRecords(ctx context.Context, ids []string) ([]string, error) {
	return s.q.RequeueAbandonedRecords(ctx, RequeueAbandonedRecordsParams{
		PendingState:   int32(outbox.PendingDelivery),
		AbandonedState: int32(outbox.MaxAttemptsReached),
		Ids:            ids,
	})
}

func secondsToDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

func toRecord(r Outbox) outbox.Record {
	return outbox.Record{
		ID: r.ID,
		Message: outbox.Message{
			Key:   r.MessageKey,
			Body:  r.Body,
			Topic: r.Topic,
		},
		State:            outbox.RecordState(r.State),
		CreatedOn:        r.CreatedOn,
		LockID:           r.LockedBy,
		LockedOn:         r.LockedOn,
		ProcessedOn:      r.ProcessedOn,
		NumberOfAttempts: int(r.NumberOfAttempts),
		LastAttemptOn:    r.LastAttemptedOn,
		Error:            r.Error,
	}
}
