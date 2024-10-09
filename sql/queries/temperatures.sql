-- name: SaveTemperature :one
INSERT INTO temperatures (
    water_temp, room_temp) 
VALUES ( $1, $2) 
RETURNING *;

-- name: FindMostRecentTemperatures :one
SELECT * FROM temperatures
ORDER BY created_at DESC;
