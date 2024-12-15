-- +goose Up
ALTER TABLE events
ALTER COLUMN event_data SET NOT NULL;

-- +goose Down
ALTER TABLE events
ALTER COLUMN event_data SET NULL;
