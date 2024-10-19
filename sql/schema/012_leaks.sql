-- +goose Up
CREATE TABLE leaks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    detected_at TIMESTAMP NOT NULL,
    cleared_at TIMESTAMP
);

-- +goose Down
DROP TABLE leaks;
