package ozone

import (
	"context"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/google/uuid"
)

const DefaultOzoneDurationMinutes = "120"

type (
	OzoneResult struct {
		ID               uuid.UUID `json:"id"`
		StartTime        time.Time `json:"start_time"`
		EndTime          time.Time `json:"end_time"`
		Running          bool      `json:"running"`
		ExpectedDuration int32     `json:"expected_duration"`
		CancelRequested  bool      `json:"cancel_requested"`
	}

	Handler struct {
		store  OzoneStore
		sensor sensor.Sensors
	}

	OzoneStore interface {
		GetLatestOzone(ctx context.Context) (database.Ozone, error)
		StartOzone(ctx context.Context, arg database.StartOzoneParams) (database.Ozone, error)
		StopOzone(ctx context.Context, id uuid.UUID) (database.Ozone, error)
	}
)
