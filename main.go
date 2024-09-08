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
	"github.com/KyleBrandon/plunger-server/services/health"
	"github.com/KyleBrandon/plunger-server/services/leaks"
	"github.com/KyleBrandon/plunger-server/services/ozone"
	"github.com/KyleBrandon/plunger-server/services/plunges"
	"github.com/KyleBrandon/plunger-server/services/pump"
	"github.com/KyleBrandon/plunger-server/services/users"
	_ "github.com/lib/pq"
)

const CONFIG_FILENAME string = "config.json"

type serverConfig struct {
	ServerPort  string
	DatabaseURL string
	Sensors     sensor.SensorConfig
	DB          *database.Queries
	JobManager  *jobs.JobConfig
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	config, err := initializeServerConfig()
	if err != nil {
		slog.Error("failed to load config file")
		os.Exit(1)
	}

	mux := http.NewServeMux()

	healthHandler := health.NewHandler()
	healthHandler.RegisterRoutes(mux)

	userHandler := users.NewHandler(config.DB)
	userHandler.RegisterRoutes(mux)

	ozoneHandler := ozone.NewHandler(config.JobManager, config.DB)
	ozoneHandler.RegisterRoutes(mux)

	leakHandler := leaks.NewHandler(config.JobManager, config.DB)
	leakHandler.RegisterRoutes(mux)
	leakHandler.StartMonitoringLeaks()

	pumpHandler := pump.NewHandler(&config.Sensors)
	pumpHandler.RegisterRoutes(mux)

	plungesHandler := plunges.NewHandler(config.DB, &config.Sensors)
	plungesHandler.RegisterRoutes(mux)

	config.runServer(mux)
}

func (config *serverConfig) runServer(mux *http.ServeMux) {
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
