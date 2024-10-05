// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package database

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	EventType int32
	EventData json.RawMessage
}

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

type Plunge struct {
	ID               uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
	StartTime        sql.NullTime
	StartWaterTemp   string
	StartRoomTemp    string
	EndTime          sql.NullTime
	EndWaterTemp     string
	EndRoomTemp      string
	Running          bool
	ExpectedDuration int32
	AvgWaterTemp     string
	AvgRoomTemp      string
}

type User struct {
	ID        uuid.UUID
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
	ApiKey    string
}
