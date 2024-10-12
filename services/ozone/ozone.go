package ozone

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/utils"
)

func NewHandler(store OzoneStore, sensor sensor.Sensors) *Handler {
	return &Handler{
		store,
		sensor,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/ozone", h.handlerOzoneGet)
	mux.HandleFunc("POST /v1/ozone/start", h.handlerOzoneStart)
	mux.HandleFunc("POST /v1/ozone/stop", h.handlerOzoneStop)
}

func databaseToOzoneResult(db database.Ozone) OzoneResult {
	o := OzoneResult{
		ID:               db.ID,
		StartTime:        db.StartTime.Time,
		EndTime:          db.EndTime.Time,
		Running:          db.Running,
		ExpectedDuration: db.ExpectedDuration,
	}

	if db.StartTime.Valid {
		o.StartTime = db.StartTime.Time
	}

	if db.EndTime.Valid {
		o.EndTime = db.EndTime.Time
	}

	if db.StatusMessage.Valid {
		o.StatusMessage = db.StatusMessage.String
	}

	return o
}

func (h *Handler) handlerOzoneGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerGetOzone")

	job, err := h.store.GetLatestOzone(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "cound not find any ozone job", err)
		return
	}

	response := databaseToOzoneResult(job)
	utils.RespondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) handlerOzoneStart(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerStartOzone")

	ozone, err := h.store.GetLatestOzone(r.Context())
	if err == nil && ozone.Running {
		utils.RespondWithError(w, http.StatusNotModified, "Ozone generator is already running", err)
		return
	}

	durationStr := r.URL.Query().Get("duration")
	if durationStr == "" {
		durationStr = DefaultOzoneDurationMinutes
	}

	duration, err := strconv.Atoi(durationStr)
	if err != nil || duration < 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid 'duration' parameter", err)
		return
	}

	startTime := sql.NullTime{
		Time:  time.Now().UTC(),
		Valid: true,
	}

	args := database.StartOzoneParams{
		StartTime:        startTime,
		ExpectedDuration: int32(duration),
	}

	ozone, err = h.store.StartOzone(r.Context(), args)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "could not start the ozone timer", err)
		return

	}

	err = h.sensor.TurnOzoneOn()
	if err != nil {
		message := fmt.Sprintf("Failed to turn on ozone generator: %v", err.Error())
		updateArgs := database.UpdateOzoneStatusParams{
			ID:            ozone.ID,
			StatusMessage: sql.NullString{String: message, Valid: true},
		}
		var dbErr error
		ozone, dbErr = h.store.UpdateOzoneStatus(r.Context(), updateArgs)
		if dbErr != nil {
			slog.Error("failed to save ozone status", "error", dbErr, "status", message)
			// return the db error to the user
			err = dbErr
		}

		utils.RespondWithError(w, http.StatusInternalServerError, "failed to start the ozone generator", err)
		return
	}

	response := databaseToOzoneResult(ozone)

	utils.RespondWithJSON(w, http.StatusCreated, response)
}

func (h *Handler) handlerOzoneStop(w http.ResponseWriter, r *http.Request) {
	slog.Info("handlerStopOzone")

	ozone, err := h.store.GetLatestOzone(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusNotModified, "could not find ozone job", err)
		return
	}

	// turn ozone off no matter what
	err = h.sensor.TurnOzoneOff()
	if err != nil {
		slog.Info("failed to turn ozone generator off", "error", err)
	}

	if !ozone.Running {
		utils.RespondWithError(w, http.StatusNotModified, "ozone not running", err)
		return
	}

	_, err = h.store.StopOzone(r.Context(), ozone.ID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotModified, "could not cancel ozone job", err)
		return
	}
	slog.Info("ozone stopped")

	utils.RespondWithNoContent(w, http.StatusNoContent)
}
