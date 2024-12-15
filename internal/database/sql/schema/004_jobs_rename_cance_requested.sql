-- +goose Up
ALTER TABLE jobs 
RENAME COLUMN cance_requested TO cancel_requested;

-- +goose Down
ALTER TABLE jobs 
RENAME COLUMN cancel_requested TO cance_requested;
