-- +goose Up
CREATE TABLE temperatures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    water_temp NUMERIC(4, 1),
    room_temp NUMERIC(4, 1)
);

-- +goose Down
DROP TABLE temperatures;
