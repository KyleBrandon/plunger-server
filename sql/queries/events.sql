-- name: CreateEvent :one
INSERT INTO events(
    event_type, event_data
) VALUES ($1, $2)
RETURNING *;


-- name: GetLatestEventByType :one
SELECT * FROM events
WHERE event_type = $1 
ORDER BY created_at DESC
LIMIT 1;

-- name: GetEventsByType :many
SELECT * FROM events
WHERE event_type = $1
ORDER BY created_at DESC
LIMIT $2;



