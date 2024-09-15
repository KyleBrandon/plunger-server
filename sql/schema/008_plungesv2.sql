-- +goose Up
ALTER TABLE plunges
ADD COLUMN running BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN expected_duration INTEGER NOT NULL DEFAULT 0,
ADD COLUMN avg_water_temp NUMERIC(4,1) NOT NULL DEFAULT 0.0,
ADD COLUMN avg_room_temp NUMERIC(4,1) NOT NULL DEFAULT 0.0;

-- +goose Down
ALTER TABLE plunges 
DROP COLUMN running,
DROP COLUMN expected_duration,
DROP COLUMN avg_water_temp,
DROP COLUMN avg_room_temp;
