-- name: GetFilters :many
SELECT * FROM filters
ORDER BY created_at DESC;

-- name: GetFilter :one
SELECT * FROM filters
WHERE id = $1;

-- name: GetLatestFilterChange :one
SELECT * FROM filters
ORDER BY created_at DESC
LIMIT 1;

-- name: ChangeFilter :one
INSERT INTO filters (changed_at, remind_at)
VALUES($1, $2)
RETURNING *;
