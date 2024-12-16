package server

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/KyleBrandon/plunger-server/config"
	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/pkg/server/filters"
	"github.com/KyleBrandon/plunger-server/pkg/server/health"
	"github.com/KyleBrandon/plunger-server/pkg/server/leaks"
	"github.com/KyleBrandon/plunger-server/pkg/server/monitor"
	"github.com/KyleBrandon/plunger-server/pkg/server/ozone"
	"github.com/KyleBrandon/plunger-server/pkg/server/plunges/v2"
	"github.com/KyleBrandon/plunger-server/pkg/server/pump"
	"github.com/KyleBrandon/plunger-server/pkg/server/status"
	"github.com/KyleBrandon/plunger-server/pkg/server/temperatures"
	"github.com/KyleBrandon/plunger-server/pkg/server/users"
	"github.com/KyleBrandon/plunger-server/pkg/utils"
	"github.com/joho/godotenv"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/twilio"
)

const (
	DEFAULT_SERVER_PORT          = "8080"
	DEFAULT_LOG_FILE_LOCATION    = "./plunger-server.log"
	DEFAULT_CONFIG_FILE_LOCATION = "./config/config.json"
)

// Used by "flag" to read command line argument
var (
	mockSensor bool
	logLevel   string
)

type ServerConfig struct {
	mux                *http.ServeMux
	mctx               *monitor.MonitorContext
	ServerPort         string
	DatabaseURL        string
	UseMockSensor      bool
	LogFileLocation    string
	ConfigFileLocation string
	Logger             *slog.Logger
	LoggerLevel        *slog.LevelVar
	LogFile            *os.File
	Notifier           *notify.Notify

	Sensors        sensor.Sensors
	Queries        *database.Queries
	DBConnection   *sql.DB
	OriginPatterns []string
}

// InitializeServer to start working
func InitializeServer() (ServerConfig, error) {
	config, err := initializeServerConfig()
	if err != nil {
		return config, err
	}

	defer config.DBConnection.Close()
	defer config.LogFile.Close()

	config.mux = http.NewServeMux()

	config.mctx = monitor.InitializeMonitorContext(config.Notifier, config.Queries, config.Sensors)

	healthHandler := health.NewHandler(config.LoggerLevel, config.Logger)
	healthHandler.RegisterRoutes(config.mux)

	temperatureHandler := temperatures.NewHandler(config.Sensors)
	temperatureHandler.RegisterRoutes(config.mux)

	userHandler := users.NewHandler(config.Queries)
	userHandler.RegisterRoutes(config.mux)

	ozoneHandler := ozone.NewHandler(config.Queries, config.Sensors, config.mctx)
	ozoneHandler.RegisterRoutes(config.mux)

	leakHandler := leaks.NewHandler(config.Queries)
	leakHandler.RegisterRoutes(config.mux)

	pumpHandler := pump.NewHandler(config.Sensors)
	pumpHandler.RegisterRoutes(config.mux)

	plungesHandler := plunges.NewHandler(config.Queries, config.Sensors)
	plungesHandler.RegisterRoutes(config.mux)

	statusHandler := status.NewHandler(config.Queries, config.Sensors, config.OriginPatterns)
	statusHandler.RegisterRoutes(config.mux)

	filterHandler := filters.NewHandler(config.Queries)
	filterHandler.RegisterRoutes(config.mux)

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	return config, err
}

// RunServer will start listening for connections
func (config *ServerConfig) RunServer() {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.ServerPort),
		Handler: config.mux,
	}

	slog.Info("Starting server", "port", config.ServerPort)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server failed", "error", err)
	}

	config.mctx.CancelAndWait()
}

func initializeServerConfig() (ServerConfig, error) {
	sc := ServerConfig{}

	// MUST BE FIRST
	sc.loadConfiguration()

	sc.configureLogger()

	// load the configuration file and environment settings
	config, err := config.LoadConfigSettings(sc.ConfigFileLocation)
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

	return sc, nil
}

func (sc *ServerConfig) configureLogger() {
	logFile, err := os.OpenFile(sc.LogFileLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Warn("Failed to open log file: %v", "error", err)
		os.Exit(1)
	}

	currentLevel := new(slog.LevelVar)

	level, err := utils.ParseLogLevel(logLevel)
	if err != nil {
		slog.Error("Failed to parse the log level, setting to DefaultLogLevel", "error", err, "log_level", logLevel)
		level = config.DefaultLogLevel
	}

	currentLevel.Set(level)

	logger := slog.New(slog.NewTextHandler(logFile,
		&slog.HandlerOptions{Level: currentLevel}))
	slog.SetDefault(logger)

	sc.Logger = logger
	sc.LoggerLevel = currentLevel
	sc.LogFile = logFile
}

func (sc *ServerConfig) loadConfiguration() {
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

	// TODO: better encapsulation and error handling
	// TODO: add the "to phone" to the user account, flesh that out
	twilioAccountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	twilioAuthToken := os.Getenv("TWILIO_AUTH_TOKEN")
	twilioFromPhone := os.Getenv("TWILIO_FROM_PHONE_NO")
	twilioToPhone := os.Getenv("TWILIO_TO_PHONE_NO")
	if len(twilioAccountSID) != 0 {
		twilioService, err := twilio.New(twilioAccountSID, twilioAuthToken, twilioFromPhone)
		if err != nil {
			log.Fatalf("failed to initialize Twilio service: %v", err)
		}

		// Set the Twilio sender phone number and recipient
		twilioService.AddReceivers(twilioToPhone) // Replace with recipient's phone number

		// Create a notifier
		notifier := notify.New()
		notifier.UseServices(twilioService)
		sc.Notifier = notifier
	}

	// mock sensor flag is a command line flag for debugging
	sc.UseMockSensor = mockSensor
}

func (config *ServerConfig) openDatabase() {
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		slog.Error("failed to open database connection", "error", err)
	}

	config.DBConnection = db
	config.Queries = database.New(db)
}

func init() {
	// initialize the mock sensor commandline flag
	flag.BoolVar(&mockSensor, "use_mock_sensor", false, "Indicate if we should use a mock sensor for the server instance.")
	flag.StringVar(&logLevel, "log_level", config.DefaultLogLevel.String(), "The log level to start the server at")
}