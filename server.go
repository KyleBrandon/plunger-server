package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/joho/godotenv"
)

const (
	DEFAULT_SERVER_PORT          = "8080"
	DEFAULT_LOG_FILE_LOCATION    = "./plunger-server.log"
	DEFAULT_CONFIG_FILE_LOCATION = "./config/config.json"
)

// Used by "flag" to read command line argument
var (
	mockSensor bool
)

type serverConfig struct {
	ServerPort         string
	DatabaseURL        string
	UseMockSensor      bool
	LogFileLocation    string
	ConfigFileLocation string
	Logger             *slog.Logger
	LoggerLevel        *slog.LevelVar

	Sensors        sensor.Sensors
	Queries        *database.Queries
	dbConnection   *sql.DB
	JobManager     jobs.JobManager
	OriginPatterns []string
}

func initializeServerConfig() (serverConfig, error) {
	sc := serverConfig{}

	// MUST BE FIRST
	sc.loadConfiguration()

	sc.configureLogger()

	// load the configuration file and environment settings
	config, err := LoadConfigSettings(sc.ConfigFileLocation)
	if err != nil {
		slog.Error("failed to load config file", "error", err)
		os.Exit(1)
	}

	// load the sensor configuration
	sensors, err := sensor.NewSensorConfig(
		config.SensorTimeoutSeconds,
		config.Devices,
		sc.UseMockSensor)
	if err != nil {
		slog.Error("failed to initailize sensors")
		os.Exit(1)
	}

	sc.Sensors = sensors
	sc.OriginPatterns = config.OriginPatterns
	sc.openDatabase()
	sc.JobManager = jobs.NewJobConfig(sc.Queries, sensors)

	return sc, nil
}

func (sc *serverConfig) configureLogger() {
	logFile, err := os.OpenFile(sc.LogFileLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	defer logFile.Close()
	currentLevel := new(slog.LevelVar)
	currentLevel.Set(DefaultLogLevel)

	logger := slog.New(slog.NewTextHandler(logFile,
		&slog.HandlerOptions{Level: currentLevel}))
	slog.SetDefault(logger)

	sc.Logger = logger
	sc.LoggerLevel = currentLevel
}

func (sc *serverConfig) loadConfiguration() {
	// load the environment
	err := godotenv.Load()
	if err != nil {
		slog.Warn("could not load .env file", "error", err)
	}

	sc.DatabaseURL = os.Getenv("DATABASE_URL")
	sc.ServerPort = os.Getenv("PORT")
	if len(sc.ServerPort) == 0 {
		sc.ServerPort = DEFAULT_SERVER_PORT
	}

	sc.LogFileLocation = os.Getenv("LOG_FILE_LOCATION")
	if len(sc.LogFileLocation) == 0 {
		sc.LogFileLocation = DEFAULT_LOG_FILE_LOCATION
	}

	sc.ConfigFileLocation = os.Getenv("CONFIG_FILE_LOCATION")
	if len(sc.ConfigFileLocation) == 0 {
		sc.ConfigFileLocation = DEFAULT_CONFIG_FILE_LOCATION
	}

	// mock sensor flag is a command line flag for debugging
	sc.UseMockSensor = mockSensor
}

func (config *serverConfig) runServer(mux *http.ServeMux, cancelMonitors context.CancelFunc) {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.ServerPort),
		Handler: mux,
	}

	slog.Info("Starting server", "port", config.ServerPort)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server failed", "error", err)
		// TODO: we should wait for the monitors to stop
		cancelMonitors()
	}
}

func (config *serverConfig) openDatabase() {
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		slog.Error("failed to open database connection", "error", err)
	}

	config.dbConnection = db
	config.Queries = database.New(db)
}

func init() {
	// initialize the mock sensor commandline flag
	flag.BoolVar(&mockSensor, "use_mock_sensor", false, "Indicate if we should use a mock sensor for the server instance.")
}
