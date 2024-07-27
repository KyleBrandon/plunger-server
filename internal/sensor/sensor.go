package sensor

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
	"github.com/yryz/ds18b20"
)

type TemperatureReading struct {
	Address      string  `json:"id"`
	TemperatureC float64 `json:"temperature_c"`
	TemperatureF float64 `json:"temperature_f"`
}

func ReadTemperatures() ([]TemperatureReading, error) {
	sensors, err := ds18b20.Sensors()
	if err != nil {
		panic(err)
	}

	readings := make([]TemperatureReading, 0, len(sensors))

	for _, sensor := range sensors {
		t, err := ds18b20.Temperature(sensor)
		if err != nil {
			log.Printf("failed to read temperatures sensor: %v\n", err)
			continue
		}

		tr := TemperatureReading{
			Address:      sensor,
			TemperatureC: t,
			TemperatureF: (t * 9 / 5) + 32,
		}

		readings = append(readings, tr)
	}

	// readLeakSensor()

	// turnPumpOff()
	// time.Sleep(10 * time.Second)
	// turnPumpOn()
	//
	// turnOzoneOn()
	// time.Sleep(10 * time.Second)
	// turnOzoneOff()
	return readings, nil
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
	time.Sleep(time.Second)
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
