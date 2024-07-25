package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type serverConfig struct {
	serverPort string
}

func main() {
	godotenv.Load()
	serverPort := os.Getenv("PORT")

	config := serverConfig{
		serverPort: serverPort,
	}

	mux := http.NewServeMux()
	// handler := config.middlewareMetricsInt(http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	// mux.Handle("/app/*", handler)

	mux.HandleFunc("GET /v1/health", config.handlerGetHealth)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.serverPort),
		Handler: mux,
	}

	fmt.Printf("Starting server on :%s\n", config.serverPort)
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server failed: %s\n", err)
	}
}
