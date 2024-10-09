package sensor

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
	"github.com/yryz/ds18b20"
)

const (
	DRIVERTYPE_DS18B20 string = "DS18B20"
	DRIVERTYPE_GPIO    string = "GPIO"
)

type SensorType int

const (
	SENSOR_TEMPERATURE string = "temperature"
	SENSOR_LEAK        string = "leak"
	SENSOR_POWER       string = "power"
)

type SensorConfig struct {
	SensorTimeout      time.Duration
	Devices            []DeviceConfig
	TemperatureSensors map[string]DeviceConfig
	LeakSensor         DeviceConfig
	OzoneDevice        DeviceConfig
	PumpDevice         DeviceConfig
}

type DeviceConfig struct {
	DriverType               string  `json:"driver_type"`
	SensorType               string  `json:"sensor_type"`
	Address                  string  `json:"address"`
	Name                     string  `json:"name"`
	Description              string  `json:"description"`
	NormallyOn               bool    `json:"normally_on,omitempty"`
	CalibrationOffsetCelsius float64 `json:"calibration_offset_celsius"`
}

type TemperatureReading struct {
	Name         string  `json:"name,omitempty"`
	Description  string  `json:"description,omitempty"`
	Address      string  `json:"address,omitempty"`
	TemperatureC float64 `json:"temperature_c,omitempty"`
	TemperatureF float64 `json:"temperature_f,omitempty"`
	Err          error   `json:"err,omitempty"`
}

type Sensors interface {
	ReadRoomAndWaterTemperature() (TemperatureReading, TemperatureReading)
	ReadTemperatures() []TemperatureReading
	IsLeakPresent() (bool, error)
	TurnOzoneOn() error
	TurnOzoneOff() error
	IsPumpOn() (bool, error)
	TurnPumpOn() error
	TurnPumpOff() error
}

func NewSensorConfig(sensorTimeout int, devices []DeviceConfig) (Sensors, error) {
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
			if d.Name == "Pump" {
				sc.PumpDevice = d
			} else if d.Name == "Ozone" {
				sc.OzoneDevice = d
			}
		}
	}

	return &sc, nil
}

func readTemperatureSensor(device *DeviceConfig) TemperatureReading {
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

func (config *SensorConfig) ReadTemperatures() []TemperatureReading {
	slog.Debug("ReadTemperatures")

	readings := make([]TemperatureReading, 0, len(config.TemperatureSensors))

	for _, device := range config.TemperatureSensors {
		tr := readTemperatureSensor(&device)
		readings = append(readings, tr)
	}

	return readings
}

func (config *SensorConfig) ReadRoomAndWaterTemperature() (TemperatureReading, TemperatureReading) {
	temperatures := config.ReadTemperatures()

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

func (config *SensorConfig) IsLeakPresent() (bool, error) {
	slog.Debug("IsLeakPresent")
	if err := rpio.Open(); err != nil {
		return false, err
	}

	defer rpio.Close()

	pinNumber, err := strconv.Atoi(config.LeakSensor.Address)
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

func (config *SensorConfig) TurnOzoneOn() error {
	return config.OzoneDevice.TurnOn()
}

func (config *SensorConfig) TurnOzoneOff() error {
	return config.OzoneDevice.TurnOff()
}

func (config *SensorConfig) IsPumpOn() (bool, error) {
	return config.PumpDevice.IsOn()
}

func (config *SensorConfig) TurnPumpOn() error {
	return config.PumpDevice.TurnOn()
}

func (config *SensorConfig) TurnPumpOff() error {
	return config.PumpDevice.TurnOff()
}

func (device *DeviceConfig) IsOn() (bool, error) {
	slog.Debug("Device.IsOn", "name", device.Name, "address", device.Address)
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

func (device *DeviceConfig) TurnOn() error {
	slog.Debug("Device.TurnOn", "name", device.Name)
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

func (device *DeviceConfig) TurnOff() error {
	slog.Debug("Device.TurnOff", "name", device.Name)
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
