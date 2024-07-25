package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type serverConfig struct {
	serverPort  string
	databaseURL string
	DB          *database.Queries
}

func main() {

	config := loadConfig()
	config.openDatabase()
	// test()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/healthz", config.handlerGetHealthz)
	mux.HandleFunc("POST /v1/users", config.handlerCreateUser)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.serverPort),
		Handler: mux,
	}

	fmt.Printf("Starting server on :%s\n", config.serverPort)
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server failed: %s\n", err)
	}
}

func loadConfig() serverConfig {
	godotenv.Load()
	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		log.Fatal("The PORT environment variable is not set")
	}

	databaseURL := os.Getenv("DB_CONN")
	if serverPort == "" {
		log.Fatal("The DB_CONN environment variable is not set")
	}
	config := serverConfig{
		serverPort:  serverPort,
		databaseURL: databaseURL,
	}

	return config
}

func (config *serverConfig) openDatabase() {
	db, err := sql.Open("postgres", config.databaseURL)
	if err != nil {
		log.Fatal("failed to open database connection", err)
	}

	config.DB = database.New(db)
}
