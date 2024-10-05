-- name: StartPlunge :one
INSERT INTO plunges (
    start_time, start_water_temp, start_room_temp, expected_duration, running) 
VALUES ( $1, $2, $3, $4, true) 
RETURNING *;

-- name: UpdatePlungeAvgTemp :one
UPDATE plunges
SET avg_water_temp = $1, avg_room_temp = $2
WHERE id = $3
RETURNING *;

-- name: StopPlunge :one
UPDATE plunges
SET end_time = $1, end_water_temp = $2, end_room_temp = $3, running = FALSE, updated_at = CURRENT_TIMESTAMP
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

