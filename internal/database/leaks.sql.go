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

const getLatestLeak = `-- name: GetLatestLeak :one
SELECT id, created_at, updated_at, detected_at, cleared_at FROM leaks
ORDER BY created_at DESC
LIMIT 1
`

func (q *Queries) GetLatestLeak(ctx context.Context) (Leak, error) {
	row := q.db.QueryRowContext(ctx, getLatestLeak)
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

const updateLeakCleared = `-- name: UpdateLeakCleared :one
UPDATE leaks
SET cleared_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, created_at, updated_at, detected_at, cleared_at
`

func (q *Queries) UpdateLeakCleared(ctx context.Context, id uuid.UUID) (Leak, error) {
	row := q.db.QueryRowContext(ctx, updateLeakCleared, id)
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