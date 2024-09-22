package plunges

import (
	"context"
	"sync"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/google/uuid"
)

type PlungeResponse struct {
	ID               uuid.UUID `json:"id,omitempty"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
	StartTime        time.Time `json:"start_time,omitempty"`
	EndTime          time.Time `json:"end_time,omitempty"`
	Running          bool      `json:"running"`
	ElapsedTime      float64   `json:"elapsed_time"`
	StartRoomTemp    string    `json:"start_room_temp,omitempty"`
	EndRoomTemp      string    `json:"end_room_temp,omitempty"`
	StartWaterTemp   string    `json:"start_water_temp,omitempty"`
	EndWaterTemp     string    `json:"end_water_temp,omitempty"`
	ExpectedDuration int32     `json:"expected_duration,omitempty"`
	AvgWaterTemp     string    `json:"average_water_temp,omitempty"`
	AvgRoomTemp      string    `json:"average_room_temp,omitempty"`
}

type PlungeStatus struct {
	ID           uuid.UUID     `json:"id,omitempty"`
	Remaining    time.Duration `json:"remaining,omitempty"`
	TotalTime    time.Duration `json:"total_time,omitempty"`
	Running      bool          `json:"running,omitempty"`
	WaterTemp    float64       `json:"water_temp,omitempty"`
	RoomTemp     float64       `json:"room_temp,omitempty"`
	AvgWaterTemp float64       `json:"average_water_temp,omitempty"`
	AvgRoomTemp  float64       `json:"average_room_temp,omitempty"`
}

type PlungeStore interface {
	GetLatestPlunge(ctx context.Context) (database.Plunge, error)
	GetPlungeByID(ctx context.Context, id uuid.UUID) (database.Plunge, error)
	GetPlunges(ctx context.Context) ([]database.Plunge, error)
	StartPlunge(ctx context.Context, arg database.StartPlungeParams) (database.Plunge, error)
	UpdatePlungeStatus(ctx context.Context, arg database.UpdatePlungeStatusParams) (database.Plunge, error)
	StopPlunge(ctx context.Context, arg database.StopPlungeParams) (database.Plunge, error)
}

// TODO: Clean up the sensor interface
type Sensors interface {
	ReadTemperatures() ([]sensor.TemperatureReading, error)
}

type Handler struct {
	store   PlungeStore
	sensors Sensors

	mu             sync.Mutex
	id             uuid.UUID
	startTime      time.Time
	stopTime       time.Time
	duration       time.Duration
	running        bool
	waterTempTotal float64
	roomTempTotal  float64
	tempReadCount  int64
}
