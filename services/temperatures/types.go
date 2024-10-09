package temperatures

import (
	"context"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
)

type (
	TemperatureReading struct {
		Name         string  `json:"name,omitempty"`
		Description  string  `json:"description,omitempty"`
		Address      string  `json:"address,omitempty"`
		TemperatureC float64 `json:"temperature_c,omitempty"`
		TemperatureF float64 `json:"temperature_f,omitempty"`
		Err          string  `json:"err,omitempty"`
	}

	TemperatureStore interface {
		FindMostRecentTemperatures(ctx context.Context) (database.Temperature, error)
		SaveTemperature(ctx context.Context, arg database.SaveTemperatureParams) (database.Temperature, error)
	}

	Handler struct {
		store   TemperatureStore
		sensors sensor.Sensors
	}
)
