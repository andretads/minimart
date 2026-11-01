//go:build integration

package postgres_test

import (
	"minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var integrationBase = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

func resetOutbox(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(t.Context(), "TRUNCATE TABLE outbox")
	require.NoError(t, err)
}

func addOutboxRow(t *testing.T, q *outboxpg.Queries, id, topic, key string, state outbox.RecordState, createdOn time.Time) {
	t.Helper()
	require.NoError(t, q.AddRecord(t.Context(), outboxpg.AddRecordParams{
		ID:         id,
		Topic:      topic,
		MessageKey: key,
		Body:       []byte("payload"),
		State:      int32(state), //nolint:gosec // RecordState is a bounded enum (0-2)
		CreatedOn:  createdOn,
	}))
}

func TestLockRecordsByStateSkipsSuccessorsOfAbandonedRows(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	defer pool.Close()

	resetOutbox(t, pool)
	q := outboxpg.New(pool)
	store := outboxpg.NewStore(pool)

	addOutboxRow(t, q, "abandoned", "claim-block", "k", outbox.MaxAttemptsReached, integrationBase)
	addOutboxRow(t, q, "successor", "claim-block", "k", outbox.PendingDelivery, integrationBase.Add(time.Second))
	addOutboxRow(t, q, "other-key", "claim-block", "other", outbox.PendingDelivery, integrationBase.Add(2*time.Second))

	require.NoError(t, store.UpdateRecordLockByState(t.Context(), "worker/1", integrationBase, outbox.PendingDelivery, 10))
	locked, err := store.GetRecordsByLockID(t.Context(), "worker/1")
	require.NoError(t, err)
	require.Len(t, locked, 1)
	assert.Equal(t, "other-key", locked[0].ID)

	requeued, err := store.RequeueAbandonedRecords(t.Context(), []string{"abandoned"})
	require.NoError(t, err)
	require.Equal(t, []string{"abandoned"}, requeued)
	require.NoError(t, store.ClearLocksByLockID(t.Context(), "worker/1"))

	require.NoError(t, store.UpdateRecordLockByState(t.Context(), "worker/1", integrationBase, outbox.PendingDelivery, 10))
	locked, err = store.GetRecordsByLockID(t.Context(), "worker/1")
	require.NoError(t, err)
	require.Len(t, locked, 3)
	assert.Equal(t, "abandoned", locked[0].ID)
	assert.Equal(t, "successor", locked[1].ID)
	assert.Equal(t, "other-key", locked[2].ID)
}

func TestClearAllLocksReclaimsDeadWorkerLocks(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err)
	defer pool.Close()

	resetOutbox(t, pool)
	q := outboxpg.New(pool)
	store := outboxpg.NewStore(pool)

	addOutboxRow(t, q, "dead-row", "takeover", "k", outbox.PendingDelivery, integrationBase)

	require.NoError(t, store.UpdateRecordLockByState(t.Context(), "worker/dead", integrationBase, outbox.PendingDelivery, 10))
	locked, err := store.GetRecordsByLockID(t.Context(), "worker/dead")
	require.NoError(t, err)
	require.Len(t, locked, 1)

	require.NoError(t, store.ClearAllLocks(t.Context()))
	locked, err = store.GetRecordsByLockID(t.Context(), "worker/dead")
	require.NoError(t, err)
	require.Empty(t, locked)

	require.NoError(t, store.UpdateRecordLockByState(t.Context(), "worker/new", integrationBase, outbox.PendingDelivery, 10))
	locked, err = store.GetRecordsByLockID(t.Context(), "worker/new")
	require.NoError(t, err)
	require.Len(t, locked, 1)
	assert.Equal(t, "dead-row", locked[0].ID)
}
