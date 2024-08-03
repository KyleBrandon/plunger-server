package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/google/uuid"
)

type OzoneJob struct {
	ID              uuid.UUID `json:"id"`
	Status          string    `json:"status"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	SecondsLeft     float64   `json:"seconds_left"`
	Result          string    `json:"result"`
	CancelRequested bool      `json:"cancel_requested"`
}

func mapFromDB(dbJob *database.Job) OzoneJob {

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

func (config *serverConfig) getJobById(jobId uuid.UUID) (*database.Job, error) {

	job, err := config.JobManager.DB.GetJobById(context.Background(), jobId)

	return &job, err
}

func (config *serverConfig) handlerGetOzoneStatus(w http.ResponseWriter, r *http.Request) {
	log.Println("handlerGetOzone")

	job, err := config.DB.GetLatestJobByType(r.Context(), jobs.JOBTYPE_OZONE_TIMER)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "cound not find any ozone job")
		return
	}

	response := mapFromDB(&job)
	respondWithJSON(w, http.StatusOK, response)
}

func (config *serverConfig) handlerStartOzone(writer http.ResponseWriter, req *http.Request) {
	log.Println("handlerStartOzone")

	job, err := config.JobManager.StartJobWithTimeout(runOzoneFunc, jobs.JOBTYPE_OZONE_TIMER, 2*time.Hour)
	if err != nil {
		respondWithError(writer, http.StatusInternalServerError, "could not start the ozone timer")
		return
	}

	response := mapFromDB(job)
	respondWithJSON(writer, http.StatusCreated, response)
}

func (config *serverConfig) handlerStopOzone(writer http.ResponseWriter, req *http.Request) {
	log.Println("handlerStopOzone")

	err := config.JobManager.CancelJob(jobs.JOBTYPE_OZONE_TIMER)
	if err != nil {
		respondWithError(writer, http.StatusNotModified, "could not stop the ozone job")
		return
	}

	respondWithNoContent(writer, http.StatusNoContent)
}

func runOzoneFunc(config *jobs.JobConfig, ctx context.Context, cancel context.CancelFunc, jobId uuid.UUID) {
	defer cancel()

	config.SensorConfig.TurnOzoneOn()

	for {
		select {

		case <-ctx.Done():
			// task was canceled or timedout
			config.SensorConfig.TurnOzoneOff()
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
