-- name: GetAvatar :one
SELECT account_id, object_key, bucket, content_type, size_bytes, version, created_at, updated_at
FROM avatars
WHERE account_id = $1;