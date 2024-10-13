package monitor

import (
	"context"

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
	GetLatestOzone(ctx context.Context) (database.Ozone, error)
	StopOzone(ctx context.Context, id uuid.UUID) (database.Ozone, error)
}
