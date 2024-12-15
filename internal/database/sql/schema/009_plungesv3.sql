
-- +goose Up
ALTER TABLE plunges ALTER COLUMN start_water_temp SET DEFAULT 0.0;
ALTER TABLE plunges ALTER COLUMN end_water_temp SET DEFAULT 0.0;

ALTER TABLE plunges ALTER COLUMN start_room_temp SET DEFAULT 0.0;
ALTER TABLE plunges ALTER COLUMN end_room_temp SET DEFAULT 0.0;

UPDATE plunges SET start_water_temp = 0.0 WHERE start_water_temp IS NULL;
UPDATE plunges SET end_water_temp = 0.0 WHERE end_water_temp IS NULL;

UPDATE plunges SET start_room_temp = 0.0 WHERE start_room_temp IS NULL;
UPDATE plunges SET end_room_temp = 0.0 WHERE end_room_temp IS NULL;

ALTER TABLE plunges ALTER COLUMN start_water_temp SET NOT NULL;
ALTER TABLE plunges ALTER COLUMN end_water_temp SET NOT NULL;

ALTER TABLE plunges ALTER COLUMN start_room_temp SET NOT NULL;
ALTER TABLE plunges ALTER COLUMN end_room_temp SET NOT NULL;

-- +goose Down
ALTER TABLE plunges ALTER COLUMN start_water_temp DROP NOT NULL;
ALTER TABLE plunges ALTER COLUMN end_water_temp DROP NOT NULL;

ALTER TABLE plunges ALTER COLUMN start_room_temp DROP NOT NULL;
ALTER TABLE plunges ALTER COLUMN end_room_temp DROP NOT NULL;
