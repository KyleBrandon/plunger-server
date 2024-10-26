-- +goose Up
CREATE TABLE filters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    remind_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE filters;