package main

import (
	"context"
	"encoding/json"
	"log"

	// "log"
	"net/http"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/google/uuid"
)

func (config *serverConfig) handlerLeakGet(w http.ResponseWriter, r *http.Request) {
	log.Println("handlerGetLeak")

	leakEvents := make([]database.Event, 0)

	filter := r.URL.Query().Get("filter")
	if filter == "current" {

		leak, err := config.DB.GetLatestEventByType(context.Background(), EVENTTYPE_LEAK)
		if err != nil {
			log.Printf("failed to read the latest leak event: %v\n", err)
			respondWithError(w, http.StatusNotFound, "could not read lead event")
			return
		}

		leakEvents = append(leakEvents, leak)

	} else {
		params := database.GetEventsByTypeParams{
			EventType: EVENTTYPE_LEAK,
			Limit:     100,
		}
		leaks, err := config.DB.GetEventsByType(r.Context(), params)
		if err != nil {
			log.Printf("failed to read the leak events: %v\n", err)
			respondWithError(w, http.StatusInternalServerError, "failed to read the leak events")
			return
		}

		leakEvents = append(leakEvents, leaks...)
	}

	response, err := BuildLeakEventsFromEvents(leakEvents)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not read the leak events")
		return
	}

	respondWithJSON(w, http.StatusOK, response)

}

func (config *serverConfig) StartMonitoringLeaks() error {
	job, err := config.JobManager.StartJob(runLeakDetectionFunc, jobs.JOBTYPE_LEAK_MONITOR)
	if err != nil {
		log.Printf("failed to start monitoring job for leaks: %v", err)
		return err
	}

	config.LeakMonitorJobId = job.ID

	return nil
}

func runLeakDetectionFunc(config *jobs.JobConfig, ctx context.Context, cancel context.CancelFunc, jobId uuid.UUID) {
	defer cancel()

	// get an initial reading
	leakState, err := config.SensorConfig.IsLeakPresent()
	if err != nil {
		log.Printf("failed to read if leak was present: %v\n", err)
	}

	for {
		leakPresent, err := config.SensorConfig.IsLeakPresent()
		if err != nil {
			log.Printf("failed to read if leak was present: %v\n", err)
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
				log.Printf("failed to encode the current leak transition: %v\n", err)
				continue
			}

			params := database.CreateEventParams{
				EventType: 1,
				EventData: eventData,
			}

			_, err = config.DB.CreateEvent(ctx, params)
			if err != nil {
				log.Printf("failed to store the initial leak event: %v\n", err)
			}

			// save the current state
			leakState = leakPresent
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
