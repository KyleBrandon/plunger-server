package temperatures

import (
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/pkg/server/monitor"
)

type (
	Handler struct {
		mctx    *monitor.MonitorContext
		sensors sensor.Sensors
	}

	TemperatureReading struct {
		Name         string  `json:"name,omitempty"`
		Description  string  `json:"description,omitempty"`
		Address      string  `json:"address,omitempty"`
		TemperatureC float64 `json:"temperature_c,omitempty"`
		TemperatureF float64 `json:"temperature_f,omitempty"`
		Err          string  `json:"err,omitempty"`
	}

	TemperatureNotifyRequest struct {
		TargetTemperature float32 `json:"temperature_target"`
	}
)
