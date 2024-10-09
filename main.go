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
	plungesV1 "github.com/KyleBrandon/plunger-server/services/plunges/v1"
	plungesV2 "github.com/KyleBrandon/plunger-server/services/plunges/v2"
	"github.com/KyleBrandon/plunger-server/services/pump"
	"github.com/KyleBrandon/plunger-server/services/status"
	"github.com/KyleBrandon/plunger-server/services/temperatures"
	"github.com/KyleBrandon/plunger-server/services/users"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const CONFIG_FILENAME string = "./config/config.json"

type serverConfig struct {
	ServerPort     string
	DatabaseURL    string
	Sensors        sensor.Sensors
	DB             *database.Queries
	JobManager     jobs.JobManager
	OriginPatterns []string
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

	temperatureHandler := temperatures.NewHandler(config.DB, config.Sensors)
	temperatureHandler.RegisterRoutes(mux)

	userHandler := users.NewHandler(config.DB)
	userHandler.RegisterRoutes(mux)

	ozoneHandler := ozone.NewHandler(config.JobManager, config.DB)
	ozoneHandler.RegisterRoutes(mux)

	leakHandler := leaks.NewHandler(config.JobManager, config.DB)
	leakHandler.RegisterRoutes(mux)
	leakHandler.StartMonitoringLeaks()

	pumpHandler := pump.NewHandler(config.Sensors)
	pumpHandler.RegisterRoutes(mux)

	plungesHandlerV1 := plungesV1.NewHandler(config.DB, config.Sensors)
	plungesHandlerV1.RegisterRoutes(mux)

	plungesHandlerV2 := plungesV2.NewHandler(config.DB, config.DB, config.Sensors)
	plungesHandlerV2.RegisterRoutes(mux)

	statusHandler := status.NewHandler(config.DB, config.DB, config.DB, config.Sensors, config.OriginPatterns)
	statusHandler.RegisterRoutes(mux)

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

	// read the database URL and serer port from the environment
	err = godotenv.Load()
	if err != nil {
		slog.Warn("could not load .env file", "error", err)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	serverPort := os.Getenv("PORT")

	sc := serverConfig{
		ServerPort:     serverPort,
		DatabaseURL:    databaseURL,
		Sensors:        sensorConfig,
		OriginPatterns: configSettings.OriginPatterns,
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
