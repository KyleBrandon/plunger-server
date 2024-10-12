package ozone

import (
	"database/sql"
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
	oj := OzoneResult{
		ID:               db.ID,
		StartTime:        db.StartTime.Time,
		EndTime:          db.EndTime.Time,
		Running:          db.Running,
		ExpectedDuration: db.ExpectedDuration,
		CancelRequested:  db.CancelRequested,
	}

	return oj
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

	job, err := h.store.StartOzone(r.Context(), args)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "could not start the ozone timer", err)
		return

	}

	err = h.sensor.TurnOzoneOn()
	if err != nil {
		// TODO: save error status

		utils.RespondWithError(w, http.StatusInternalServerError, "failed to start the ozone generator", err)
		return
	}

	response := databaseToOzoneResult(job)
	utils.RespondWithJSON(w, http.StatusCreated, response)
}

func (h *Handler) handlerOzoneStop(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerStopOzone")

	ozone, err := h.store.GetLatestOzone(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusNotModified, "could not find ozone job", err)
		return
	}

	// turn ozone off no matter what
	err = h.sensor.TurnOzoneOff()
	if err != nil {
		slog.Error("failed to turn ozone generator off", "error", err)
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

	utils.RespondWithNoContent(w, http.StatusNoContent)
}
