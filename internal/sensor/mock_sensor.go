package sensor

import (
	"log/slog"
)

func (m *MockSensors) readTemperatureSensor(device *DeviceConfig) TemperatureReading {
	tr := TemperatureReading{
		Name:        device.Name,
		Description: device.Description,
		Address:     device.Address,
	}

	// TODO: read from config?
	t := 10.0

	t += device.CalibrationOffsetCelsius
	tr.TemperatureC = t
	tr.TemperatureF = (t * 9 / 5) + 32
	tr.Err = nil

	return tr
}

func (m *MockSensors) ReadTemperatures() []TemperatureReading {
	slog.Debug(">>ReadTemperatures")
	defer slog.Debug("<<ReadTemperatures")

	readings := make([]TemperatureReading, 0, len(m.config.TemperatureSensors))

	for _, device := range m.config.TemperatureSensors {
		tr := m.readTemperatureSensor(&device)
		readings = append(readings, tr)
	}

	return readings
}

func (m *MockSensors) ReadRoomAndWaterTemperature() (TemperatureReading, TemperatureReading) {
	slog.Debug(">>ReadRoomAndWaterTemperature")
	defer slog.Debug("<<ReadRoomAndWaterTemperature")
	temperatures := m.ReadTemperatures()

	var waterTemp TemperatureReading
	var roomTemp TemperatureReading

	for _, temp := range temperatures {
		switch temp.Name {
		case "Room":
			roomTemp = temp
		case "Water":
			waterTemp = temp
		}
	}

	return roomTemp, waterTemp
}

func (m *MockSensors) IsLeakPresent() (bool, error) {
	slog.Debug(">>IsLeakPresent")
	defer slog.Debug("<<IsLeakPresent")

	// TODO: read from config

	return false, nil
}

func (m *MockSensors) TurnOzoneOn() error {
	slog.Debug(">>TurnOzoneOn")
	defer slog.Debug("<<TurnOzoneOn")

	// TODO: read from config
	return nil
}

func (m *MockSensors) TurnOzoneOff() error {
	slog.Debug(">>TurnOzoneOff")
	defer slog.Debug("<<TurnOzoneOff")

	// TODO: read from config
	return nil
}

func (m *MockSensors) IsPumpOn() (bool, error) {
	slog.Debug(">>IsPumpOn")
	defer slog.Debug("<<IsPumpOn")

	// TODO: read from config
	return true, nil
}

func (m *MockSensors) TurnPumpOn() error {
	slog.Debug(">>TurnPumpOn")
	defer slog.Debug("<<TurnPumpOn")

	// TODO: read from config
	return nil
}

func (m *MockSensors) TurnPumpOff() error {
	slog.Debug(">>TurnPumpOff")
	defer slog.Debug("<<TurnPumpOff")

	// TODO: read from config
	return nil
}
