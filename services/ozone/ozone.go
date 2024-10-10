package ozone

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/KyleBrandon/plunger-server/utils"
	"github.com/google/uuid"
)

func NewHandler(manager jobs.JobManager, store jobs.JobStore) *Handler {
	return &Handler{
		manager: manager,
		store:   store,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/ozone", h.handlerOzoneGet)
	mux.HandleFunc("POST /v1/ozone/start", h.handlerOzoneStart)
	mux.HandleFunc("POST /v1/ozone/stop", h.handlerOzoneStop)
}

func databaseJobToOzoneJob(dbJob *database.Job) OzoneJob {
	var status string
	var timeLeft float64
	if dbJob.Status == jobs.JOBSTATUS_STARTED {
		status = "Running"
		timeLeft = dbJob.EndTime.Sub(time.Now().UTC()).Seconds()
	} else {
		status = "Stopped"
		timeLeft = 0.0
	}

	oj := OzoneJob{
		ID:              dbJob.ID,
		Status:          status,
		StartTime:       dbJob.StartTime,
		EndTime:         dbJob.EndTime,
		SecondsLeft:     timeLeft,
		Result:          dbJob.Result.String,
		CancelRequested: dbJob.CancelRequested,
	}

	return oj
}

func (h *Handler) handlerOzoneGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerGetOzone")

	job, err := h.store.GetLatestJobByType(r.Context(), jobs.JOBTYPE_OZONE_TIMER)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "cound not find any ozone job", err)
		return
	}

	response := databaseJobToOzoneJob(&job)
	utils.RespondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) handlerOzoneStart(writer http.ResponseWriter, _ *http.Request) {
	slog.Debug("handlerStartOzone")

	job, err := h.manager.StartJobWithTimeout(runOzoneFunc, jobs.JOBTYPE_OZONE_TIMER, 2*time.Hour)
	if err != nil {
		utils.RespondWithError(writer, http.StatusInternalServerError, "could not start the ozone timer", err)
		return
	}

	response := databaseJobToOzoneJob(job)
	utils.RespondWithJSON(writer, http.StatusCreated, response)
}

func (h *Handler) handlerOzoneStop(writer http.ResponseWriter, _ *http.Request) {
	slog.Debug("handlerStopOzone")

	err := h.manager.CancelJob(jobs.JOBTYPE_OZONE_TIMER)
	if err != nil {
		utils.RespondWithError(writer, http.StatusNotModified, "could not stop the ozone job", err)
		return
	}

	utils.RespondWithNoContent(writer, http.StatusNoContent)
}

func runOzoneFunc(config *jobs.JobConfig, ctx context.Context, cancel context.CancelFunc, jobId uuid.UUID) {
	defer cancel()

	config.SensorConfig.TurnOzoneOn()

	for {
		select {

		case <-ctx.Done():
			slog.Info("Ozone finished")
			// task was canceled or timedout
			config.SensorConfig.TurnOzoneOff()
			config.StopJob(jobs.JOBTYPE_OZONE_TIMER, "Success")

			return

		case <-time.After(5 * time.Second):
			// check to see if the task was canceled by the user
			cancelRequested := config.IsJobCanceled(jobId)
			if cancelRequested {
				slog.Info("Ozone canceled by user")
				config.SensorConfig.TurnOzoneOff()
				cancel()
				continue
			}

		}
	}
}
