package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/services/filters"
	"github.com/KyleBrandon/plunger-server/services/health"
	"github.com/KyleBrandon/plunger-server/services/leaks"
	"github.com/KyleBrandon/plunger-server/services/monitor"
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
	Queries        *database.Queries
	dbConnection   *sql.DB
	JobManager     jobs.JobManager
	OriginPatterns []string
	logger         *slog.Logger
}

var sensorType bool

func main() {
	flag.Parse() // Parse the command-line flags

	config, err := initializeServerConfig()
	if err != nil {
		slog.Error("failed to load config file")
		os.Exit(1)
	}

	defer config.dbConnection.Close()

	mux := http.NewServeMux()

	monitorHandler := monitor.NewHandler(config.Queries, config.Sensors)
	ctx, cancelMonitors := context.WithCancel(context.Background())

	monitorHandler.StartMonitorRoutines(ctx)

	healthHandler := health.NewHandler(config.logger)
	healthHandler.RegisterRoutes(mux)

	temperatureHandler := temperatures.NewHandler(config.Sensors)
	temperatureHandler.RegisterRoutes(mux)

	userHandler := users.NewHandler(config.Queries)
	userHandler.RegisterRoutes(mux)

	ozoneHandler := ozone.NewHandler(config.Queries, config.Sensors)
	ozoneHandler.RegisterRoutes(mux)

	leakHandler := leaks.NewHandler(config.Queries)
	leakHandler.RegisterRoutes(mux)

	pumpHandler := pump.NewHandler(config.Sensors)
	pumpHandler.RegisterRoutes(mux)

	plungesHandlerV1 := plungesV1.NewHandler(config.Queries, config.Sensors)
	plungesHandlerV1.RegisterRoutes(mux)

	plungesHandlerV2 := plungesV2.NewHandler(config.Queries, config.Sensors)
	plungesHandlerV2.RegisterRoutes(mux)

	statusHandler := status.NewHandler(config.Queries, config.Sensors, config.OriginPatterns)
	statusHandler.RegisterRoutes(mux)

	filterHandler := filters.NewHandler(config.Queries)
	filterHandler.RegisterRoutes(mux)

	config.runServer(mux, cancelMonitors)
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

func initializeServerConfig() (serverConfig, error) {
	sc := serverConfig{}

	sc.logger = health.ConfigureLogger()

	configSettings, err := LoadConfigFile(CONFIG_FILENAME)
	if err != nil {
		slog.Error("failed to load config file", "error", err)
		os.Exit(1)
	}

	databaseURL, serverPort, useMockSensor := readParameters()

	sensorConfig, err := sensor.NewSensorConfig(configSettings.SensorTimeoutSeconds, configSettings.Devices, useMockSensor)
	if err != nil {
		slog.Error("failed to initailize sensors")
		os.Exit(1)
	}

	sc.ServerPort = serverPort
	sc.DatabaseURL = databaseURL
	sc.Sensors = sensorConfig
	sc.OriginPatterns = configSettings.OriginPatterns

	sc.openDatabase()
	sc.JobManager = jobs.NewJobConfig(sc.Queries, sensorConfig)

	return sc, nil
}

func (config *serverConfig) openDatabase() {
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		slog.Error("failed to open database connection", "error", err)
	}

	config.dbConnection = db
	config.Queries = database.New(db)
}

// TODO: refactor this, it's messy
func readParameters() (string, string, bool) {
	// read the database URL and serer port from the environment
	err := godotenv.Load()
	if err != nil {
		slog.Warn("could not load .env file", "error", err)
	}

	serverPort := ""
	databaseURL := ""

	slog.Info("sensor type", "use_mock_sensor", sensorType)

	if len(serverPort) == 0 {
		serverPort = os.Getenv("PORT")
	}

	if len(databaseURL) == 0 {
		databaseURL = os.Getenv("DATABASE_URL")
	}

	return databaseURL, serverPort, sensorType
}

func init() {
	flag.BoolVar(&sensorType, "use_mock_sensor", false, "Indicate if we should use a mock sensor for the server instance.")
}
