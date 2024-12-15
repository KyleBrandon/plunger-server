package main

import (
	"flag"
	"log/slog"
	"os"

	_ "net/http/pprof"

	"github.com/KyleBrandon/plunger-server/pkg/server"
	_ "github.com/lib/pq"
)

func main() {
	// parse the command-line flags
	flag.Parse()

	config, err := server.InitializeServer()
	if err != nil {
		slog.Error("failed to load config file")
		os.Exit(1)
	}

	// start the server
	config.RunServer()
}
