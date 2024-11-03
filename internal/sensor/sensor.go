package sensor

import (
	"log/slog"
	"time"
)

func NewSensorConfig(sensorTimeout int, devices []DeviceConfig, useMockSensor bool) (Sensors, error) {
	slog.Debug("NewSensorConfig")
	sc := SensorConfig{
		SensorTimeout: time.Duration(sensorTimeout) * time.Second,
		Devices:       devices,
	}

	sc.TemperatureSensors = make(map[string]DeviceConfig)
	for _, d := range sc.Devices {
		switch d.SensorType {
		case SENSOR_TEMPERATURE:
			if d.DriverType == DRIVERTYPE_DS18B20 {
				sc.TemperatureSensors[d.Address] = d
			}
		case SENSOR_LEAK:
			sc.LeakSensor = d

		case SENSOR_POWER:
			switch d.Name {
			case "Pump":
				sc.PumpDevice = d
			case "Ozone":
				sc.OzoneDevice = d
			}
		}
	}

	if useMockSensor {
		return &MockSensors{config: sc}, nil
	}

	return &HardwareSensors{config: sc}, nil
}
