package sensor

import "time"

const (
	DRIVERTYPE_DS18B20 string = "DS18B20"
	DRIVERTYPE_GPIO    string = "GPIO"
	SENSOR_TEMPERATURE string = "temperature"
	SENSOR_LEAK        string = "leak"
	SENSOR_POWER       string = "power"
)

type (
	SensorType int

	SensorConfig struct {
		SensorTimeout time.Duration
		Devices       []DeviceConfig

		TemperatureSensors map[string]DeviceConfig
		LeakSensor         DeviceConfig
		OzoneDevice        DeviceConfig
		PumpDevice         DeviceConfig
	}

	DeviceConfig struct {
		DriverType               string  `json:"driver_type"`
		SensorType               string  `json:"sensor_type"`
		Address                  string  `json:"address"`
		Name                     string  `json:"name"`
		Description              string  `json:"description"`
		NormallyOn               bool    `json:"normally_on,omitempty"`
		CalibrationOffsetCelsius float64 `json:"calibration_offset_celsius"`
	}

	TemperatureReading struct {
		Name         string  `json:"name,omitempty"`
		Description  string  `json:"description,omitempty"`
		Address      string  `json:"address,omitempty"`
		TemperatureC float64 `json:"temperature_c,omitempty"`
		TemperatureF float64 `json:"temperature_f,omitempty"`
		Err          error   `json:"err,omitempty"`
	}

	Sensors interface {
		ReadRoomAndWaterTemperature() (TemperatureReading, TemperatureReading)
		ReadTemperatures() []TemperatureReading
		IsLeakPresent() (bool, error)
		TurnOzoneOn() error
		TurnOzoneOff() error
		IsPumpOn() (bool, error)
		TurnPumpOn() error
		TurnPumpOff() error
	}

	HardwareSensors struct {
		config SensorConfig
	}

	MockSensors struct {
		config SensorConfig
	}
)
