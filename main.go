package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

const CONFIG_FILENAME string = "config.json"

type serverConfig struct {
	ServerPort       string
	DatabaseURL      string
	Sensors          sensor.SensorConfig
	DB               *database.Queries
	JobManager       *jobs.JobConfig
	LeakMonitorJobId uuid.UUID
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	config, err := initializeServerConfig()
	if err != nil {
		slog.Error("failed to load config file")
		os.Exit(1)
	}

	config.StartMonitoringLeaks()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", config.handlerHealthGet)
	mux.HandleFunc("POST /v1/users", config.handlerUserCreate)
	mux.HandleFunc("GET /v1/users", config.handlerUserGet)
	mux.HandleFunc("GET /v1/temperatures", config.handlerTemperaturesGet)
	mux.HandleFunc("GET /v1/ozone", config.handlerOzoneGet)
	mux.HandleFunc("POST /v1/ozone/start", config.handlerOzoneStart)
	mux.HandleFunc("POST /v1/ozone/stop", config.handlerOzoneStop)
	mux.HandleFunc("GET /v1/leaks", config.handlerLeakGet)
	mux.HandleFunc("GET /v1/pump", config.handlerPumpGet)
	mux.HandleFunc("POST /v1/pump/start", config.handlerPumpStart)
	mux.HandleFunc("POST /v1/pump/stop", config.handlerPumpStop)
	mux.HandleFunc("GET /v1/plunges", config.handlePlungesGet)
	mux.HandleFunc("GET /v1/plunges/{PLUNGE_ID}", config.handlePlungesGet)
	mux.HandleFunc("POST /v1/plunges", config.handlePlungesStart)
	mux.HandleFunc("PUT /v1/plunges/{PLUNGE_ID}", config.handlePlungesStop)

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
		slog.Error("failed to load config file", "error", err)
		os.Exit(1)
	}

	sensorConfig, err := sensor.NewSensorConfig(configSettings.SensorTimeoutSeconds, configSettings.Devices)
	if err != nil {
		slog.Error("failed to initailize sensors")
		os.Exit(1)
	}

	sc := serverConfig{
		ServerPort:  configSettings.ServerPort,
		DatabaseURL: configSettings.DatabaseURL,
		Sensors:     sensorConfig,
	}

	sc.openDatabase()
	sc.JobManager = jobs.NewJobConfig(sc.DB, sensorConfig)

	return sc, nil
}

func (config *serverConfig) openDatabase() {
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		slog.Error("failed to open database connection", "error", err)
	}

	config.DB = database.New(db)
}
