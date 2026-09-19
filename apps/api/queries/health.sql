-- name: GetSystemMeta :one
SELECT key, val, created_at
FROM _system_meta
WHERE key = $1 LIMIT 1;
