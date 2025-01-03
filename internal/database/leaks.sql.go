// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: leaks.sql

package database

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const clearDetectedLeak = `-- name: ClearDetectedLeak :one
UPDATE leaks
SET cleared_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, created_at, updated_at, detected_at, cleared_at
`

func (q *Queries) ClearDetectedLeak(ctx context.Context, id uuid.UUID) (Leak, error) {
	row := q.db.QueryRowContext(ctx, clearDetectedLeak, id)
	var i Leak
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DetectedAt,
		&i.ClearedAt,
	)
	return i, err
}

const createLeakDetected = `-- name: CreateLeakDetected :one
INSERT INTO leaks (detected_at)
VALUES ($1)
RETURNING id, created_at, updated_at, detected_at, cleared_at
`

func (q *Queries) CreateLeakDetected(ctx context.Context, detectedAt time.Time) (Leak, error) {
	row := q.db.QueryRowContext(ctx, createLeakDetected, detectedAt)
	var i Leak
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DetectedAt,
		&i.ClearedAt,
	)
	return i, err
}

const getLatestLeakDetected = `-- name: GetLatestLeakDetected :one
SELECT id, created_at, updated_at, detected_at, cleared_at FROM leaks
ORDER BY created_at DESC
LIMIT 1
`

func (q *Queries) GetLatestLeakDetected(ctx context.Context) (Leak, error) {
	row := q.db.QueryRowContext(ctx, getLatestLeakDetected)
	var i Leak
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DetectedAt,
		&i.ClearedAt,
	)
	return i, err
}
