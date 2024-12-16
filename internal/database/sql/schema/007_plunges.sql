-- +goose Up
CREATE TABLE plunges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    start_time TIMESTAMP,
    start_water_temp NUMERIC(4, 1),
    start_room_temp NUMERIC(4, 1),
    end_time TIMESTAMP,
    end_water_temp NUMERIC(4, 1),
    end_room_temp NUMERIC(4, 1)
);

-- +goose Down
DROP TABLE plunges;
