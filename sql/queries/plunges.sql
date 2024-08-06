-- name: StartPlunge :one
INSERT INTO plunges (
    start_time, start_water_temp, start_room_temp
) VALUES ( $1, $2, $3)
RETURNING *;

-- name: StopPlunge :one
UPDATE plunges 
SET end_time = $1, end_water_temp = $2, end_room_temp = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $4
RETURNING *;

-- name: GetLatestPlunge :one
SELECT * FROM plunges 
ORDER BY created_at DESC
LIMIT 1;

-- name: GetPlunges :many
SELECT * FROM plunges 
ORDER BY created_at DESC;

-- name: GetPlungeByID :one
SELECT * FROM plunges
WHERE id = $1;

