package main

import (
	"encoding/json"
	"io"
	"os"

	"github.com/KyleBrandon/plunger-server/internal/sensor"
)

type Config struct {
	ServerPort           string                `json:"server_port"`
	DatabaseURL          string                `json:"database_url"`
	Devices              []sensor.DeviceConfig `json:"devices"`
	SensorTimeoutSeconds int                   `json:"sensor_timeout_seconds"`
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
