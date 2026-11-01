-- name: UpsertAvatar :one
INSERT INTO avatars (account_id, object_key, bucket, content_type, size_bytes, version, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, 1, $6, $7)
ON CONFLICT (account_id) DO UPDATE SET
  object_key = EXCLUDED.object_key,
  bucket = EXCLUDED.bucket,
  content_type = EXCLUDED.content_type,
  size_bytes = EXCLUDED.size_bytes,
  version = avatars.version + 1,
  updated_at = EXCLUDED.updated_at
RETURNING account_id, object_key, bucket, content_type, size_bytes, version, created_at, updated_at;