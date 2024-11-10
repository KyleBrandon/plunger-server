package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"

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
	_ "github.com/lib/pq"
)

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

	healthHandler := health.NewHandler(config.LoggerLevel, config.Logger)
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
