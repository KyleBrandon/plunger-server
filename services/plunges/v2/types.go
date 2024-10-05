package plunges

import (
	"context"
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
	StartWaterTemp   string    `json:"start_water_temp,omitempty"`
	StartRoomTemp    string    `json:"start_room_temp,omitempty"`
	EndTime          time.Time `json:"end_time,omitempty"`
	EndWaterTemp     string    `json:"end_water_temp,omitempty"`
	EndRoomTemp      string    `json:"end_room_temp,omitempty"`
	Running          bool      `json:"running"`
	ExpectedDuration int32     `json:"expected_duration,omitempty"`
	AvgWaterTemp     string    `json:"average_water_temp,omitempty"`
	AvgRoomTemp      string    `json:"average_room_temp,omitempty"`
}

type PlungeStore interface {
	GetLatestPlunge(ctx context.Context) (database.Plunge, error)
	GetPlungeByID(ctx context.Context, id uuid.UUID) (database.Plunge, error)
	GetPlunges(ctx context.Context) ([]database.Plunge, error)
	StartPlunge(ctx context.Context, arg database.StartPlungeParams) (database.Plunge, error)
	UpdatePlungeAvgTemp(ctx context.Context, arg database.UpdatePlungeAvgTempParams) (database.Plunge, error)
	StopPlunge(ctx context.Context, arg database.StopPlungeParams) (database.Plunge, error)
}

type Handler struct {
	store   PlungeStore
	sensors sensor.Sensors
}
