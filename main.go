package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/KyleBrandon/plunger-server/internal/database"
	_ "github.com/lib/pq"
)

const CONFIG_FILENAME string = "config.json"

type serverConfig struct {
	ServerPort  string
	DatabaseURL string
	Sensors     []SensorConfig
	Devices     []DeviceConfig
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
	mux.HandleFunc("GET /v1/temperatures/{location}", config.handlerGetTemperatureByLocation)

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

	sc := serverConfig{
		ServerPort:  configSettings.ServerPort,
		DatabaseURL: configSettings.DatabaseURL,
		Sensors:     configSettings.Sensors,
		Devices:     configSettings.Devices,
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
