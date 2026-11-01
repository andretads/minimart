-- name: AddRecord :exec
INSERT INTO outbox (id, topic, message_key, body, state, created_on)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: LockRecordsByState :exec
WITH claimed AS (
    SELECT id
    FROM outbox
    WHERE outbox.state = $1 AND outbox.locked_by IS NULL
      AND NOT EXISTS (
          SELECT 1
          FROM outbox blocker
          WHERE blocker.state = 2 -- MaxAttemptsReached
            AND blocker.topic = outbox.topic
            AND blocker.message_key = outbox.message_key
            AND (blocker.created_on < outbox.created_on
                 OR (blocker.created_on = outbox.created_on AND blocker.id < outbox.id))
      )
    ORDER BY outbox.created_on, outbox.id
    LIMIT $2
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox AS o
SET locked_by = $3, locked_on = $4
FROM claimed
WHERE o.id = claimed.id;

-- name: GetRecordsByLockID :many
SELECT id, topic, message_key, body, state, created_on, locked_by, locked_on,
       processed_on, number_of_attempts, last_attempted_on, error
FROM outbox
WHERE locked_by = $1
ORDER BY created_on, id;

-- name: UpdateRecord :exec
UPDATE outbox
SET state = $2,
    processed_on = $3,
    locked_by = $4,
    locked_on = $5,
    number_of_attempts = $6,
    last_attempted_on = $7,
    error = $8
WHERE id = $1;

-- name: ClearLocksByLockID :exec
UPDATE outbox
SET locked_by = NULL, locked_on = NULL
WHERE locked_by = $1;

-- name: ClearAllLocks :exec
UPDATE outbox
SET locked_by = NULL, locked_on = NULL
WHERE locked_by IS NOT NULL;

-- name: ClearStaleLocks :exec
UPDATE outbox
SET locked_by = NULL, locked_on = NULL
WHERE locked_by IS NOT NULL AND locked_on < $1;

-- name: DeleteDeliveredRecordsBefore :exec
DELETE FROM outbox
WHERE state = $1 AND created_on < $2;

-- name: GetOutboxStats :one
SELECT
    count(*) FILTER (WHERE state = 0 AND locked_by IS NULL)::bigint AS pending,
    count(*) FILTER (WHERE state = 0 AND locked_by IS NOT NULL)::bigint AS locked,
    count(*) FILTER (WHERE state = 1)::bigint AS delivered,
    count(*) FILTER (WHERE state = 2)::bigint AS abandoned,
    COALESCE(EXTRACT(EPOCH FROM (now() - (min(created_on) FILTER (WHERE state = 0 AND locked_by IS NULL)))), 0)::double precision AS oldest_pending_seconds,
    COALESCE(EXTRACT(EPOCH FROM (now() - (min(locked_on) FILTER (WHERE state = 0 AND locked_by IS NOT NULL)))), 0)::double precision AS oldest_locked_seconds,
    COALESCE(EXTRACT(EPOCH FROM (now() - (min(created_on) FILTER (WHERE state = 1)))), 0)::double precision AS oldest_delivered_seconds,
    COALESCE(EXTRACT(EPOCH FROM (now() - (min(created_on) FILTER (WHERE state = 2)))), 0)::double precision AS oldest_abandoned_seconds;

-- name: ListAbandonedRecords :many
SELECT id, topic, message_key, created_on
FROM outbox
WHERE state = $1
ORDER BY created_on, id
LIMIT $2;

-- name: ListAbandonedRecordsByTopic :many
SELECT id, topic, message_key, created_on
FROM outbox
WHERE state = $1 AND topic = $2
ORDER BY created_on, id
LIMIT $3;

-- name: RequeueAbandonedRecords :many
WITH picked AS (
    SELECT id
    FROM outbox
    WHERE outbox.state = sqlc.arg(abandoned_state)
      AND outbox.id = ANY(sqlc.arg(ids)::text[])
    FOR UPDATE SKIP LOCKED
)
UPDATE outbox AS o
SET state = sqlc.arg(pending_state),
    locked_by = NULL,
    locked_on = NULL,
    processed_on = NULL,
    number_of_attempts = 0,
    last_attempted_on = NULL,
    error = NULL
FROM picked
WHERE o.id = picked.id
RETURNING o.id;

-- name: TryAcquireRelayLock :one
SELECT pg_try_advisory_lock(hashtext(sqlc.arg(namespace)), sqlc.arg(key)) AS acquired;

-- name: ReleaseRelayLock :one
SELECT pg_advisory_unlock(hashtext(sqlc.arg(namespace)), sqlc.arg(key)) AS released;
