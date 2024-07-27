package main

import (
	"encoding/json"
	"io"
	"os"
)

type Config struct {
	ServerPort  string         `json:"server_port"`
	DatabaseURL string         `json:"database_url"`
	Sensors     []SensorConfig `json:"sensors"`
	Devices     []DeviceConfig `json:"devices"`
}

type DriverType int

const (
	DS18B20 DriverType = iota // 1-wire
	GPIO
)

type SensorType int

const (
	Temperature SensorType = iota
	Leak
)

type SensorConfig struct {
	DriverType  DriverType `json:"driver_type"`
	SensorType  SensorType `json:"sensor_type"`
	Address     string     `json:"address"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
}

type DeviceConfig struct {
	DriverType  DriverType `json:"driver_type"`
	Address     string     `json:"address"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
}

func LoadConfigFile(filename string) (Config, error) {
	var config Config

	file, err := os.Open(filename)
	if err != nil {
		return config, err
	}

	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return config, err
	}

	return config, nil

}
