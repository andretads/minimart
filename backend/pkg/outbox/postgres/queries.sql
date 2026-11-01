-- name: AddRecord :exec
INSERT INTO outbox (id, topic, message_key, body, state, created_on)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: LockRecordsByState :exec
UPDATE outbox
SET locked_by = $1, locked_on = $2
WHERE state = $3 AND locked_by IS NULL;

-- name: GetRecordsByLockID :many
SELECT id, topic, message_key, body, state, created_on, locked_by, locked_on,
       processed_on, number_of_attempts, last_attempted_on, error
FROM outbox
WHERE locked_by = $1
ORDER BY created_on;

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

-- name: ClearStaleLocks :exec
UPDATE outbox
SET locked_by = NULL, locked_on = NULL
WHERE locked_by IS NOT NULL AND locked_on < $1;

-- name: DeleteRecordsBefore :exec
DELETE FROM outbox
WHERE state <> $1 AND created_on < $2;
