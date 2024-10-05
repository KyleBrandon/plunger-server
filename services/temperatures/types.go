package temperatures

import "github.com/KyleBrandon/plunger-server/internal/sensor"

type TemperatureReading struct {
	Name         string  `json:"name,omitempty"`
	Description  string  `json:"description,omitempty"`
	Address      string  `json:"address,omitempty"`
	TemperatureC float64 `json:"temperature_c,omitempty"`
	TemperatureF float64 `json:"temperature_f,omitempty"`
	Err          string  `json:"err,omitempty"`
}

type Handler struct {
	sensors sensor.Sensors
}
