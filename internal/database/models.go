// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package database

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Job struct {
	ID              uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	JobType         int32
	Status          int32
	StartTime       time.Time
	EndTime         time.Time
	Result          sql.NullString
	CancelRequested bool
}

type User struct {
	ID        uuid.UUID
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
	ApiKey    string
}
