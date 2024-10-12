-- name: StartOzone :one
INSERT INTO ozone (
    start_time, running, expected_duration
) VALUES ( $1, true, $2)
RETURNING *;

-- name: StopOzone :one
UPDATE ozone
SET end_time = CURRENT_TIMESTAMP, running = FALSE, cancel_requested = TRUE
WHERE id = $1
RETURNING *;

-- name: GetLatestOzone :one
SELECT * FROM ozone
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdateOzoneStatus :one
UPDATE ozone
SET status_message = $1
WHERE id = $2
RETURNING *;
