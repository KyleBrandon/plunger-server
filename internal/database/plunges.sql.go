// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: plunges.sql

package database

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const getLatestPlunge = `-- name: GetLatestPlunge :one
SELECT id, created_at, updated_at, start_time, start_water_temp, start_room_temp, end_time, end_water_temp, end_room_temp, running, expected_duration, avg_water_temp, avg_room_temp FROM plunges 
ORDER BY created_at DESC
LIMIT 1
`

func (q *Queries) GetLatestPlunge(ctx context.Context) (Plunge, error) {
	row := q.db.QueryRowContext(ctx, getLatestPlunge)
	var i Plunge
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.StartTime,
		&i.StartWaterTemp,
		&i.StartRoomTemp,
		&i.EndTime,
		&i.EndWaterTemp,
		&i.EndRoomTemp,
		&i.Running,
		&i.ExpectedDuration,
		&i.AvgWaterTemp,
		&i.AvgRoomTemp,
	)
	return i, err
}

const getPlungeByID = `-- name: GetPlungeByID :one
SELECT id, created_at, updated_at, start_time, start_water_temp, start_room_temp, end_time, end_water_temp, end_room_temp, running, expected_duration, avg_water_temp, avg_room_temp FROM plunges
WHERE id = $1
`

func (q *Queries) GetPlungeByID(ctx context.Context, id uuid.UUID) (Plunge, error) {
	row := q.db.QueryRowContext(ctx, getPlungeByID, id)
	var i Plunge
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.StartTime,
		&i.StartWaterTemp,
		&i.StartRoomTemp,
		&i.EndTime,
		&i.EndWaterTemp,
		&i.EndRoomTemp,
		&i.Running,
		&i.ExpectedDuration,
		&i.AvgWaterTemp,
		&i.AvgRoomTemp,
	)
	return i, err
}

const getPlunges = `-- name: GetPlunges :many
SELECT id, created_at, updated_at, start_time, start_water_temp, start_room_temp, end_time, end_water_temp, end_room_temp, running, expected_duration, avg_water_temp, avg_room_temp FROM plunges 
ORDER BY created_at DESC
`

func (q *Queries) GetPlunges(ctx context.Context) ([]Plunge, error) {
	rows, err := q.db.QueryContext(ctx, getPlunges)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Plunge
	for rows.Next() {
		var i Plunge
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.StartTime,
			&i.StartWaterTemp,
			&i.StartRoomTemp,
			&i.EndTime,
			&i.EndWaterTemp,
			&i.EndRoomTemp,
			&i.Running,
			&i.ExpectedDuration,
			&i.AvgWaterTemp,
			&i.AvgRoomTemp,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const startPlunge = `-- name: StartPlunge :one
INSERT INTO plunges (
    start_time, start_water_temp, start_room_temp, expected_duration, running) 
VALUES ( $1, $2, $3, $4, true) 
RETURNING id, created_at, updated_at, start_time, start_water_temp, start_room_temp, end_time, end_water_temp, end_room_temp, running, expected_duration, avg_water_temp, avg_room_temp
`

type StartPlungeParams struct {
	StartTime        sql.NullTime
	StartWaterTemp   string
	StartRoomTemp    string
	ExpectedDuration int32
}

func (q *Queries) StartPlunge(ctx context.Context, arg StartPlungeParams) (Plunge, error) {
	row := q.db.QueryRowContext(ctx, startPlunge,
		arg.StartTime,
		arg.StartWaterTemp,
		arg.StartRoomTemp,
		arg.ExpectedDuration,
	)
	var i Plunge
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.StartTime,
		&i.StartWaterTemp,
		&i.StartRoomTemp,
		&i.EndTime,
		&i.EndWaterTemp,
		&i.EndRoomTemp,
		&i.Running,
		&i.ExpectedDuration,
		&i.AvgWaterTemp,
		&i.AvgRoomTemp,
	)
	return i, err
}

const stopPlunge = `-- name: StopPlunge :one
UPDATE plunges 
SET end_time = $1, end_water_temp = $2, end_room_temp = $3, avg_water_temp = $4, avg_room_temp = $5, running = FALSE, updated_at = CURRENT_TIMESTAMP
WHERE id = $6
RETURNING id, created_at, updated_at, start_time, start_water_temp, start_room_temp, end_time, end_water_temp, end_room_temp, running, expected_duration, avg_water_temp, avg_room_temp
`

type StopPlungeParams struct {
	EndTime      sql.NullTime
	EndWaterTemp string
	EndRoomTemp  string
	AvgWaterTemp string
	AvgRoomTemp  string
	ID           uuid.UUID
}

func (q *Queries) StopPlunge(ctx context.Context, arg StopPlungeParams) (Plunge, error) {
	row := q.db.QueryRowContext(ctx, stopPlunge,
		arg.EndTime,
		arg.EndWaterTemp,
		arg.EndRoomTemp,
		arg.AvgWaterTemp,
		arg.AvgRoomTemp,
		arg.ID,
	)
	var i Plunge
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.StartTime,
		&i.StartWaterTemp,
		&i.StartRoomTemp,
		&i.EndTime,
		&i.EndWaterTemp,
		&i.EndRoomTemp,
		&i.Running,
		&i.ExpectedDuration,
		&i.AvgWaterTemp,
		&i.AvgRoomTemp,
	)
	return i, err
}

const updatePlungeStatus = `-- name: UpdatePlungeStatus :one
UPDATE plunges
SET avg_water_temp = $1, avg_room_temp = $2
WHERE id = $3
RETURNING id, created_at, updated_at, start_time, start_water_temp, start_room_temp, end_time, end_water_temp, end_room_temp, running, expected_duration, avg_water_temp, avg_room_temp
`

type UpdatePlungeStatusParams struct {
	AvgWaterTemp string
	AvgRoomTemp  string
	ID           uuid.UUID
}

func (q *Queries) UpdatePlungeStatus(ctx context.Context, arg UpdatePlungeStatusParams) (Plunge, error) {
	row := q.db.QueryRowContext(ctx, updatePlungeStatus, arg.AvgWaterTemp, arg.AvgRoomTemp, arg.ID)
	var i Plunge
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.StartTime,
		&i.StartWaterTemp,
		&i.StartRoomTemp,
		&i.EndTime,
		&i.EndWaterTemp,
		&i.EndRoomTemp,
		&i.Running,
		&i.ExpectedDuration,
		&i.AvgWaterTemp,
		&i.AvgRoomTemp,
	)
	return i, err
}
