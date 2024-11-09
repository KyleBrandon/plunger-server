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
	slog.Debug(">>handlerGetOzone")
	defer slog.Debug("<<handlerGetOzone")

	job, err := h.store.GetLatestOzoneEntry(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "cound not find any ozone job", err)
		return
	}

	response := databaseToOzoneResult(job)
	utils.RespondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) handlerOzoneStart(w http.ResponseWriter, r *http.Request) {
	slog.Info(">>handlerStartOzone")
	defer slog.Info("<<handlerStartOzone")

	ozone, err := h.store.GetLatestOzoneEntry(r.Context())
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

	args := database.StartOzoneGeneratorParams{
		StartTime:        startTime,
		ExpectedDuration: int32(duration),
	}

	ozone, err = h.store.StartOzoneGenerator(r.Context(), args)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "could not start the ozone timer", err)
		return

	}

	err = h.sensor.TurnOzoneOn()
	if err != nil {
		message := fmt.Sprintf("Failed to turn on ozone generator: %v", err.Error())
		updateArgs := database.UpdateOzoneEntryStatusParams{
			ID:            ozone.ID,
			StatusMessage: sql.NullString{String: message, Valid: true},
		}
		var dbErr error
		ozone, dbErr = h.store.UpdateOzoneEntryStatus(r.Context(), updateArgs)
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
	slog.Info(">>handlerStopOzone")
	defer slog.Info("<<handlerStopOzone")

	ozone, err := h.store.GetLatestOzoneEntry(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusNotModified, "could not find ozone job", err)
		return
	}

	// turn ozone off no matter what
	err = h.sensor.TurnOzoneOff()
	if err != nil {
		slog.Error("failed to turn ozone generator off", "error", err)

		// Update the ozone status to indicate it was not turned off
		arg := database.UpdateOzoneEntryStatusParams{
			ID:            ozone.ID,
			StatusMessage: sql.NullString{Valid: true, String: "Failed to turn ozone generator off"},
		}
		_, err = h.store.UpdateOzoneEntryStatus(r.Context(), arg)
		if err != nil {
			// we wee unable to update the ozone status, log an error at a minimum
			slog.Error("failed to update ozone status to indicate ozone was not turned off", "error", err)
		}
	}

	// if the ozone is not running then return
	if !ozone.Running {
		utils.RespondWithError(w, http.StatusNotModified, "ozone not running", err)
		return
	}

	// update the database and stop the ozone
	_, err = h.store.StopOzoneGenerator(r.Context(), ozone.ID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotModified, "could not cancel ozone job", err)
		return
	}

	utils.RespondWithNoContent(w, http.StatusNoContent)
}
