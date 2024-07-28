package sensor

import (
	"fmt"
	"log"
	"os"
	"sync"
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
}

type DeviceConfig struct {
	DriverType  string `json:"driver_type"`
	SensorType  string `json:"sensor_type"`
	Address     string `json:"address"`
	Name        string `json:"name"`
	Description string `json:"description"`
	NormallyOn  bool   `json:"normally_on,omitempty"`
}

type TemperatureReading struct {
	Name         string  `json:"name,omitempty"`
	Description  string  `json:"description,omitempty"`
	Address      string  `json:"address,omitempty"`
	TemperatureC float64 `json:"temperature_c,omitempty"`
	TemperatureF float64 `json:"temperature_f,omitempty"`
	Err          error   `json:"err,omitempty"`
}

func NewSensorConfig(sensorTimeout int, devices []DeviceConfig) (SensorConfig, error) {
	sc := SensorConfig{
		SensorTimeout: time.Duration(sensorTimeout) * time.Second,
		Devices:       devices,
	}

	sc.TemperatureSensors = make(map[string]DeviceConfig)
	for _, d := range sc.Devices {
		if d.SensorType != SENSOR_TEMPERATURE || d.DriverType != DRIVERTYPE_DS18B20 {
			continue
		}

		sc.TemperatureSensors[d.Address] = d
	}

	return sc, nil
}

func readTemperatureSensor(device *DeviceConfig, wg *sync.WaitGroup, readings chan<- TemperatureReading) {
	defer wg.Done()

	t, err := ds18b20.Temperature(device.Address)

	tr := TemperatureReading{
		Name:         device.Name,
		Description:  device.Description,
		Address:      device.Address,
		TemperatureC: t,
		TemperatureF: (t * 9 / 5) + 32,
		Err:          err,
	}

	readings <- tr
}

func (config *SensorConfig) ReadTemperatures() ([]TemperatureReading, error) {

	var wg sync.WaitGroup
	wg.Add(len(config.TemperatureSensors))
	readings := make(chan TemperatureReading, len(config.TemperatureSensors))

	for _, device := range config.TemperatureSensors {
		go readTemperatureSensor(&device, &wg, readings)
	}

	wg.Wait()
	close(readings)

	var err error = nil
	results := make([]TemperatureReading, 0, len(readings))
	for reading := range readings {
		if reading.Err != nil {
			log.Printf("failed to read sensor (%v): %v\n", reading.Address, reading.Err)
			err = reading.Err
		}
		results = append(results, reading)
	}

	return results, err
}

func readLeakSensor() {
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	defer rpio.Close()

	pin := rpio.Pin(17)
	pin.Input()
	for i := 0; i < 10; i++ {
		res := pin.Read()
		leakDetected := false
		if res == 1 {
			leakDetected = true
		}

		fmt.Printf("Leak detected: %v\n", leakDetected)
		time.Sleep(time.Second)
	}
}

func turnPumpOn() {
	log.Println("turn pump on")
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	defer rpio.Close()

	pin := rpio.Pin(22)
	pin.Output()
	pin.Low()

}

func turnPumpOff() {
	log.Println("turn pump off")
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	defer rpio.Close()

	pin := rpio.Pin(22)
	pin.Output()
	pin.High()
}

func turnOzoneOn() {
	log.Println("turn ozone on")
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	defer rpio.Close()

	pin := rpio.Pin(24)
	pin.Output()
	pin.High()

}

func turnOzoneOff() {
	log.Println("turn ozone off")
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	defer rpio.Close()

	pin := rpio.Pin(24)
	pin.Output()
	pin.Low()
}

func readPowerRelays() {
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	defer rpio.Close()

	pin := rpio.Pin(22)
	pin.Output()
	fmt.Println("toggle outlet 1 on")
	pin.High()
	time.Sleep(5 * time.Second)
	fmt.Println("toggle outlet 1 off")
	pin.Low()

	time.Sleep(time.Second)

	pin = rpio.Pin(23)
	pin.Output()
	fmt.Println("toggle outlet 2 on")
	pin.High()
	time.Sleep(time.Second)
	fmt.Println("toggle outlet 2 off")
	pin.Low()
	time.Sleep(time.Second)

	pin = rpio.Pin(24)
	pin.Output()
	fmt.Println("toggle outlet 3 on")
	pin.High()
	time.Sleep(time.Second)
	fmt.Println("toggle outlet 3 off")
	pin.Low()
	time.Sleep(time.Second)

}
