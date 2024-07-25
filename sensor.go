package main

import (
	"fmt"
	"github.com/stianeikeland/go-rpio/v4"
	"github.com/yryz/ds18b20"
	"os"
	"time"
)

/*
#cgo CXXFLAGS: -std=c++11
#cgo LDFLAGS: -lstdc++
#include "./src/dht-sensor.h"
*/
//import "C"

//func initialize() int {
//	return int(C.initialize())
//}
//
//func readDHT11() {
// godht.initialize()
//}

func test() {

	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	defer rpio.Close()

	//readLeakSensor()
	readTemperature()
	//readPowerRelays()

}

func readTemperature() {
	sensors, err := ds18b20.Sensors()
	if err != nil {
		panic(err)
	}

	fmt.Printf("sensor IDs: %v\n", sensors)

	for _, sensor := range sensors {
		t, err := ds18b20.Temperature(sensor)
		if err == nil {
			t = (t * 9 / 5) + 32
			fmt.Printf("sensor: %s temperature: %.2f°F\n", sensor, t)
		} else {
			fmt.Printf("err: %v\n", err)
		}
	}
}

func readLeakSensor() {
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

func readPowerRelays() {
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

	pin = rpio.Pin(25)
	pin.Output()
	fmt.Println("toggle outlet 4 on")
	pin.High()
	time.Sleep(time.Second)
	fmt.Println("toggle outlet 4 off")
	pin.Low()
	time.Sleep(time.Second)
}
