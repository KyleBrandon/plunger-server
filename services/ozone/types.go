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
		StatusMessage    string    `json:"status_message"`
	}

	Handler struct {
		store  OzoneStore
		sensor sensor.Sensors
	}

	OzoneStore interface {
		GetLatestOzoneEntry(ctx context.Context) (database.Ozone, error)
		StartOzoneGenerator(ctx context.Context, arg database.StartOzoneGeneratorParams) (database.Ozone, error)
		StopOzoneGenerator(ctx context.Context, id uuid.UUID) (database.Ozone, error)
		UpdateOzoneEntryStatus(ctx context.Context, arg database.UpdateOzoneEntryStatusParams) (database.Ozone, error)
	}
)
