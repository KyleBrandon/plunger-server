package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
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
	}
	timeLeft = 0.0

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

func (config *serverConfig) handlerGetOzone(writer http.ResponseWriter, req *http.Request) {
	log.Println("handlerGetOzone")

	var job *database.Job
	var err error

	param := req.PathValue("JOBID")
	if param != "" {
		jobId, err := uuid.Parse(param)
		if err != nil {
			log.Printf("invalid job id: %v", param)
			respondWithError(writer, http.StatusNotFound, "could not find job")
			return
		}

		job, err = config.getJobById(jobId)
		if err != nil {
			respondWithError(writer, http.StatusNotFound, "could not find job by id")
			return
		}
	} else {
		job, err = config.JobManager.GetRunningJob(jobs.JOBTYPE_OZONE_TIMER)
		if err != nil {
			respondWithError(writer, http.StatusNotFound, "could not find a running ozone job")
			return
		}
	}

	response := mapFromDB(job)
	respondWithJSON(writer, http.StatusOK, response)
}

func (config *serverConfig) handlerStartOzone(writer http.ResponseWriter, req *http.Request) {
	log.Println("handlerStartOzone")

	job, err := config.JobManager.StartJob(runOzoneFunc, jobs.JOBTYPE_OZONE_TIMER, 2*time.Hour)
	if err != nil {
		respondWithError(writer, http.StatusInternalServerError, "could not start the ozone timer")
		return
	}

	response := mapFromDB(job)
	respondWithJSON(writer, http.StatusCreated, response)
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
