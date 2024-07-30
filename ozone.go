package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/google/uuid"
)

type OzoneJob struct {
	ID              uuid.UUID
	Status          string
	StartTime       time.Time
	EndTime         time.Time
	Result          string
	CancelRequested bool
}

func (config *serverConfig) handlerGetOzone(writer http.ResponseWriter, req *http.Request) {
	log.Println("handlerGetOzone")

}

func (config *serverConfig) handlerStartOzone(writer http.ResponseWriter, req *http.Request) {
	log.Println("handlerStartOzone")

	_, err := config.JobManager.StartJob(runOzoneFunc, jobs.JOBTYPE_OZONE_TIMER, time.Minute)
	if err != nil {
		respondWithError(writer, http.StatusInternalServerError, "could not start the ozone timer")
		return
	}

	respondWithNoContent(writer, http.StatusCreated)
}

func (config *serverConfig) handlerStopOzone(writer http.ResponseWriter, req *http.Request) {
	log.Println("handlerStopOzone")
	jobConfig := jobs.NewJobConfig(config.DB)

	err := jobConfig.CancelJob(jobs.JOBTYPE_OZONE_TIMER)
	if err != nil {
		respondWithError(writer, http.StatusNotModified, "could not stop the ozone job")
		return
	}

	respondWithNoContent(writer, http.StatusNoContent)
}

func runOzoneFunc(config *jobs.JobConfig, ctx context.Context, cancel context.CancelFunc, jobId uuid.UUID) {
	defer cancel()

	sensor.TurnOzoneOn()

	for {
		select {

		case <-ctx.Done():
			// task was canceled or timedout
			sensor.TurnOzoneOff()
			config.StopJob(jobs.JOBTYPE_OZONE_TIMER, "Success")

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
