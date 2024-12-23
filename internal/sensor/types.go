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
		// ReadRoomAndWaterTemperature will read just the temperature sensors named 'Room' and 'Water'.
		ReadRoomAndWaterTemperature() (TemperatureReading, TemperatureReading)
		// ReadTemperatures will read all temperature sensors.
		//  returns a slice of `TemperatureReading`
		ReadTemperatures() []TemperatureReading
		// IsLeakPresent will determine the leak sensor detects water
		IsLeakPresent() (bool, error)
		// TurnOzoneOn will start the ozone generator.
		TurnOzoneOn() error
		// TurnOzoneOff will stop the ozone generator.
		//  returns error if the sensor could not be read.
		TurnOzoneOff() error
		// IsPumpOn will check if the pump currently has power.
		//  returns true if there is power to the pump
		//  returns false if t here is no power to the pump
		//  returns error if the sensor could not be read.
		IsPumpOn() (bool, error)
		// TurnPumpOn will turn power on to the pump.
		//  returns error if the sensor could not be read.
		TurnPumpOn() error
		// TurnPumpOff will turn power off to the pump.
		//  returns error if the sensor could not be read.
		TurnPumpOff() error
	}

	HardwareSensors struct {
		config SensorConfig
	}

	MockSensors struct {
		config SensorConfig
	}
)
