package plunges

import (
	"context"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/google/uuid"
)

const DefaultPlungeDurationSeconds = "180"

type (
	PlungeResponse struct {
		ID               uuid.UUID `json:"id"`
		CreatedAt        time.Time `json:"created_at"`
		UpdatedAt        time.Time `json:"updated_at"`
		StartTime        time.Time `json:"start_time"`
		StartWaterTemp   string    `json:"start_water_temp"`
		StartRoomTemp    string    `json:"start_room_temp"`
		EndTime          time.Time `json:"end_time"`
		EndWaterTemp     string    `json:"end_water_temp"`
		EndRoomTemp      string    `json:"end_room_temp"`
		Running          bool      `json:"running"`
		ExpectedDuration int32     `json:"expected_duration"`
		AvgWaterTemp     string    `json:"average_water_temp"`
		AvgRoomTemp      string    `json:"average_room_temp"`
	}

	PlungeStore interface {
		FindMostRecentTemperatures(ctx context.Context) (database.Temperature, error)
		GetLatestPlunge(ctx context.Context) (database.Plunge, error)
		GetPlungeByID(ctx context.Context, id uuid.UUID) (database.Plunge, error)
		GetPlunges(ctx context.Context) ([]database.Plunge, error)
		StartPlunge(ctx context.Context, arg database.StartPlungeParams) (database.Plunge, error)
		UpdatePlungeAvgTemp(ctx context.Context, arg database.UpdatePlungeAvgTempParams) (database.Plunge, error)
		StopPlunge(ctx context.Context, arg database.StopPlungeParams) (database.Plunge, error)
	}

	Handler struct {
		store   PlungeStore
		sensors sensor.Sensors
	}
)
