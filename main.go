package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	_ "github.com/lib/pq"
)

const CONFIG_FILENAME string = "config.json"

type serverConfig struct {
	ServerPort  string
	DatabaseURL string
	Sensors     sensor.SensorConfig
	DB          *database.Queries
}

func main() {

	config, err := initializeServerConfig()
	if err != nil {
		log.Fatal("failed to load config file")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", config.handlerGetHealth)
	mux.HandleFunc("POST /v1/users", config.handlerCreateUser)
	mux.HandleFunc("GET /v1/users", config.handlerGetUser)
	mux.HandleFunc("GET /v1/temperatures", config.handlerGetTemperatures)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.ServerPort),
		Handler: mux,
	}

	fmt.Printf("Starting server on :%s\n", config.ServerPort)
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server failed: %s\n", err)
	}
}

func initializeServerConfig() (serverConfig, error) {
	configSettings, err := LoadConfigFile(CONFIG_FILENAME)
	if err != nil {
		log.Fatal("failed to load config file")
	}

	sensorConfig, err := sensor.NewSensorConfig(configSettings.SensorTimeoutSeconds, configSettings.Devices)
	if err != nil {
		log.Fatal("failed to initailize sensors")
	}

	sc := serverConfig{
		ServerPort:  configSettings.ServerPort,
		DatabaseURL: configSettings.DatabaseURL,
		Sensors:     sensorConfig,
	}

	sc.openDatabase()

	return sc, nil
}

func (config *serverConfig) openDatabase() {
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		log.Fatal("failed to open database connection", err)
	}

	config.DB = database.New(db)
}
