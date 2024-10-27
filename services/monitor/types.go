package monitor

import (
	"context"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/google/uuid"
)

type Handler struct {
	store   MonitorStore
	sensors sensor.Sensors
}

type MonitorStore interface {
	SaveTemperature(ctx context.Context, arg database.SaveTemperatureParams) (database.Temperature, error)
	GetLatestOzoneEntry(ctx context.Context) (database.Ozone, error)
	StopOzoneGenerator(ctx context.Context, id uuid.UUID) (database.Ozone, error)
	UpdateOzoneEntryStatus(ctx context.Context, args database.UpdateOzoneEntryStatusParams) (database.Ozone, error)
	GetLatestLeakDetected(ctx context.Context) (database.Leak, error)
	CreateLeakDetected(ctx context.Context, detectedAt time.Time) (database.Leak, error)
	ClearDetectedLeak(ctx context.Context, id uuid.UUID) (database.Leak, error)
}
