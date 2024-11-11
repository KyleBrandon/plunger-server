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

type Filter struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	ChangedAt time.Time
	RemindAt  time.Time
}

type Leak struct {
	ID         uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DetectedAt time.Time
	ClearedAt  sql.NullTime
}

type Ozone struct {
	ID               uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
	StartTime        sql.NullTime
	EndTime          sql.NullTime
	Running          bool
	ExpectedDuration int32
	StatusMessage    sql.NullString
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

type Temperature struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	WaterTemp sql.NullString
	RoomTemp  sql.NullString
}

type User struct {
	ID        uuid.UUID
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
	ApiKey    string
}
