package leaks

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/KyleBrandon/plunger-server/utils"
	"github.com/google/uuid"
)

func BuildLeakEventsFromEvents(events []database.Event) ([]LeakEvent, error) {
	leakEvents := make([]LeakEvent, 0, len(events))

	for _, event := range events {
		var dbLeakEvent DbLeakEvent
		err := json.Unmarshal(event.EventData, &dbLeakEvent)
		if err != nil {
			slog.Error("failed to deserialize the leak event", "error", err)
			return nil, err
		}

		leakEvent := LeakEvent{
			UpdatedAt:    dbLeakEvent.EventTime,
			LeakDetected: dbLeakEvent.CurrentState,
		}

		leakEvents = append(leakEvents, leakEvent)
	}

	return leakEvents, nil
}

func NewHandler(manager jobs.JobManager, store LeakStore) *Handler {
	h := Handler{}
	h.store = store
	h.manager = manager

	return &h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/leaks", h.handlerLeakGet)
}

func (h *Handler) StartMonitoringLeaks() error {
	slog.Debug("StartMonitoringLeaks")

	job, err := h.manager.StartJob(runLeakDetectionFunc, jobs.JOBTYPE_LEAK_MONITOR)
	if err != nil {
		slog.Error("failed to start monitoring job for leaks", "error", err)
		return err
	}

	h.leakMonitorJobId = job.ID

	return nil
}

func (h *Handler) handlerLeakGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerGetLeak")

	leakEvents := make([]database.Event, 0)

	filter := r.URL.Query().Get("filter")
	// TODO: Break this up and have a separate handler for one leak vs multiple
	if filter == "current" {

		leak, err := h.store.GetLatestEventByType(context.Background(), EVENTTYPE_LEAK)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "could not read lead event", err)
			return
		}

		leakEvents = append(leakEvents, leak)

	} else {
		params := database.GetEventsByTypeParams{
			EventType: EVENTTYPE_LEAK,
			Limit:     100,
		}
		leaks, err := h.store.GetEventsByType(r.Context(), params)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to read the leak events", err)
			return
		}

		leakEvents = append(leakEvents, leaks...)
	}

	response, err := BuildLeakEventsFromEvents(leakEvents)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "could not read the leak events", err)
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, response)
}

func runLeakDetectionFunc(config *jobs.JobConfig, ctx context.Context, cancel context.CancelFunc, jobId uuid.UUID) {
	defer cancel()

	// get an initial reading
	leakState, err := config.SensorConfig.IsLeakPresent()
	if err != nil {
		slog.Warn("failed to read if leak was present", "error", err)
	}

	for {
		leakPresent, err := config.SensorConfig.IsLeakPresent()
		if err != nil {
			slog.Warn("failed to read if leak was present", "error", err)
		}

		// we only persist when a transition occurs
		if leakPresent != leakState {
			leakData := DbLeakEvent{
				EventTime:     time.Now().UTC(),
				PreviousState: leakState,
				CurrentState:  leakPresent,
			}

			eventData, err := json.Marshal(leakData)
			if err != nil {
				slog.Warn("failed to encode the current leak transition", "error", err)
				continue
			}

			params := database.CreateEventParams{
				EventType: 1,
				EventData: eventData,
			}

			_, err = config.DB.CreateEvent(ctx, params)
			if err != nil {
				slog.Warn("failed to store the initial leak event", "error", err)
			}

			// save the current state
			leakState = leakPresent
		}

		// If there is a leak, turn off the pump.
		// TODO: this should be an event that we have listeners on
		if leakPresent {
			config.SensorConfig.TurnPumpOff()
		}

		select {

		case <-ctx.Done():
			// task was canceled or timedout
			config.StopJob(jobs.JOBTYPE_LEAK_MONITOR, "Success")

			return

		case <-time.After(5 * time.Second):
			// check to see if the task was canceled by the user
			cancelRequested := config.IsJobCanceled(jobId)
			if cancelRequested {
				cancel()
				continue
			}

		}
	}
}
