package temperatures

import "github.com/KyleBrandon/plunger-server/internal/sensor"

type Sensors interface {
	ReadTemperatures() ([]sensor.TemperatureReading, error)
}

type Handler struct {
	sensors Sensors
}
