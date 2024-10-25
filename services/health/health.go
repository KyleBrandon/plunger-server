package health

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/KyleBrandon/plunger-server/utils"
)

const DefaultLogLevel = slog.LevelInfo

var current_level *slog.LevelVar

func NewHandler(logger *slog.Logger) *Handler {
	h := Handler{}
	h.logger = logger
	h.level = DefaultLogLevel
	return &h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/health", h.handlerHealthGet)
	mux.HandleFunc("GET /v1/logger", h.handlerLoggerGet)
	mux.HandleFunc("PUT /v1/logger", h.handlerLoggerUpdate)
}

func (h *Handler) handlerHealthGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("enter handlerGetHealth")
	response := struct {
		Status string `json:"status"`
	}{
		Status: "ok",
	}

	utils.RespondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) handlerLoggerGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>handlerLoggerGet")
	defer slog.Debug("<<handlerLoggerGet")

	h.mu.Lock()
	defer h.mu.Unlock()
	slog.Info("Current log level", "level", h.level.String())

	response := struct {
		LogLevel string `json:"log_level"`
	}{
		LogLevel: fmt.Sprintf("Current log level: %s", h.level.String()),
	}

	utils.RespondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) handlerLoggerUpdate(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>handlerLoggerUpdate")
	defer slog.Debug("<<handlerLoggerUpdate")

	h.mu.Lock()
	defer h.mu.Unlock()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	defer r.Body.Close()

	request := struct {
		LogLevel string `json:"log_level"`
	}{}

	if err := json.Unmarshal(body, &request); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	level, err := parseLogLevel(request.LogLevel)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid log level", err)
		return
	}

	current_level.Set(level)

	utils.RespondWithNoContent(w, http.StatusOK)
}

func parseLogLevel(logLevel string) (slog.Level, error) {
	switch strings.ToLower(logLevel) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	case "fatal":
		return slog.LevelError, nil // No fatal in slog, map to error.
	default:
		return slog.LevelInfo, fmt.Errorf("invalid log level: %s", logLevel)
	}
}

func ConfigureLogger() *slog.Logger {
	// TODO: don't like a global
	current_level = new(slog.LevelVar)
	current_level.Set(DefaultLogLevel)

	logger := slog.New(slog.NewTextHandler(os.Stderr,
		&slog.HandlerOptions{Level: current_level}))
	slog.SetDefault(logger)

	return logger
}
