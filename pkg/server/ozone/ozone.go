package ozone

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/pkg/server/monitor"
	"github.com/KyleBrandon/plunger-server/pkg/utils"
)

func NewHandler(store OzoneStore, sensor sensor.Sensors, mctx *monitor.MonitorContext) *Handler {
	return &Handler{
		store,
		sensor,
		mctx,
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

// handlerOzoneStart will trigger the ozone generator to start producing ozone and log the start in the database.
func (h *Handler) handlerOzoneStart(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>handlerStartOzone")
	defer slog.Debug("<<handlerStartOzone")

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

	h.mctx.OzoneCh <- monitor.OzoneTask{Action: monitor.OZONEACTION_START, Duration: duration}

	utils.RespondWithNoContent(w, http.StatusCreated)
}

// handlerOzoneStop will trigger the ozone generator to stop producing ozone and log the duration of the run.
func (h *Handler) handlerOzoneStop(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>handlerStopOzone")
	defer slog.Debug("<<handlerStopOzone")

	h.mctx.OzoneCh <- monitor.OzoneTask{Action: monitor.OZONEACTION_STOP}

	utils.RespondWithNoContent(w, http.StatusNoContent)
}
