package sensor

import (
	"log/slog"
	"strconv"

	"github.com/stianeikeland/go-rpio"
	"github.com/yryz/ds18b20"
)

func (s *HardwareSensors) readTemperatureSensor(device *DeviceConfig) TemperatureReading {
	tr := TemperatureReading{
		Name:        device.Name,
		Description: device.Description,
		Address:     device.Address,
	}

	t, err := ds18b20.Temperature(device.Address)
	if err != nil {
		slog.Error("failed to read sensor", "name", device.Name, "address", device.Address, "error", err)
		tr.Err = err
	} else {

		t += device.CalibrationOffsetCelsius
		tr.TemperatureC = t
		tr.TemperatureF = (t * 9 / 5) + 32
		tr.Err = nil
	}

	return tr
}

func (s *HardwareSensors) ReadTemperatures() []TemperatureReading {
	slog.Debug(">>ReadRoomAndWaterTemperature")
	defer slog.Debug("<<ReadRoomAndWaterTemperature")

	readings := make([]TemperatureReading, 0, len(s.config.TemperatureSensors))

	for _, device := range s.config.TemperatureSensors {
		tr := s.readTemperatureSensor(&device)
		readings = append(readings, tr)
	}

	return readings
}

func (s *HardwareSensors) ReadRoomAndWaterTemperature() (TemperatureReading, TemperatureReading) {
	temperatures := s.ReadTemperatures()

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

func (s *HardwareSensors) IsLeakPresent() (bool, error) {
	slog.Debug(">>IsLeakPresent")
	defer slog.Debug("<<IsLeakPresent")

	if err := rpio.Open(); err != nil {
		return false, err
	}

	defer rpio.Close()

	pinNumber, err := strconv.Atoi(s.config.LeakSensor.Address)
	if err != nil {
		return false, err
	}

	pin := rpio.Pin(pinNumber)
	res := pin.Read()
	if res == 1 {
		return true, nil
	}

	return false, nil
}

func (s *HardwareSensors) TurnOzoneOn() error {
	slog.Info(">>TurnOzoneOn")
	defer slog.Info("<<TurnOzoneOn")

	slog.Info("ozone device", "config", s.config.OzoneDevice)

	err := turnDeviceOn(&s.config.OzoneDevice)
	if err != nil {
		slog.Error("failed to turn ozone generator on", "error", err)
		return err
	}

	deviceOn, err := isDeviceOn(&s.config.OzoneDevice)
	slog.Info("ozone on", "err", err, "result", deviceOn)

	return err
}

func (s *HardwareSensors) TurnOzoneOff() error {
	slog.Info(">>TurnOzoneOff")
	defer slog.Info("<<TurnOzoneOff")

	return turnDeviceOff(&s.config.OzoneDevice)
}

func (s *HardwareSensors) IsPumpOn() (bool, error) {
	slog.Debug(">>IsPumpOn")
	defer slog.Debug("<<IsPumpOn")

	return isDeviceOn(&s.config.PumpDevice)
}

func (s *HardwareSensors) TurnPumpOn() error {
	slog.Debug(">>TurnPumpOn")
	defer slog.Debug("<<TurnPumpOn")

	return turnDeviceOn(&s.config.PumpDevice)
}

func (s *HardwareSensors) TurnPumpOff() error {
	slog.Debug(">>TurnPumpOff")
	defer slog.Debug("<<TurnPumpOff")

	return turnDeviceOff(&s.config.PumpDevice)
}

func isDeviceOn(device *DeviceConfig) (bool, error) {
	slog.Debug(">>isDeviceOn", "name", device.Name, "address", device.Address)
	defer slog.Debug("<<isDeviceOn")

	if err := rpio.Open(); err != nil {
		return false, err
	}

	defer rpio.Close()

	pinNumber, err := strconv.Atoi(device.Address)
	if err != nil {
		return false, err
	}

	pin := rpio.Pin(pinNumber)
	res := pin.Read()

	var pinOnValue rpio.State = 1
	if device.NormallyOn {
		pinOnValue = 0
	}

	if res == pinOnValue {
		return true, nil
	}

	return false, nil
}

func turnDeviceOn(device *DeviceConfig) error {
	slog.Info(">>turnDeviceOn", "name", device.Name)
	defer slog.Info("<<turnDeviceOn", "name", device.Name)

	if err := rpio.Open(); err != nil {
		return err
	}

	defer rpio.Close()

	pinNumber, err := strconv.Atoi(device.Address)
	if err != nil {
		return err
	}

	pin := rpio.Pin(pinNumber)
	pin.Output()

	// if the device is normally on, that means the pin is low when it is on
	if device.NormallyOn {
		pin.Low()
	} else {
		pin.High()
	}

	return nil
}

func turnDeviceOff(device *DeviceConfig) error {
	slog.Debug(">>turnDeviceOff", "name", device.Name)
	defer slog.Debug("<<turnDeviceOff", "name", device.Name)

	if err := rpio.Open(); err != nil {
		return err
	}

	defer rpio.Close()

	pinNumber, err := strconv.Atoi(device.Address)
	if err != nil {
		return err
	}

	pin := rpio.Pin(pinNumber)
	pin.Output()

	// if the device is normally on, that means the pin is high when it is off
	if device.NormallyOn {
		pin.High()
	} else {
		pin.Low()
	}

	return nil
}
