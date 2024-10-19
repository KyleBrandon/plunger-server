-- name: GetLatestLeak :one
SELECT * FROM leaks
ORDER BY created_at DESC
LIMIT 1;

-- name: CreateLeakDetected :one
INSERT INTO leaks (detected_at)
VALUES ($1)
RETURNING *;

-- name: UpdateLeakCleared :one
UPDATE leaks
SET cleared_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;
