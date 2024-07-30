package main

import (
	"log"
	"net/http"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/google/uuid"
)

func (config *serverConfig) handlerGetOzone(writer http.ResponseWriter, req *http.Request) {
	log.Println("handlerGetOzone")
}

func (config *serverConfig) handlerStartOzone(writer http.ResponseWriter, req *http.Request) {
	log.Println("handlerStartOzone")

	jobConfig := jobs.NewJobConfig(config.DB, jobs.JOBTYPE_OZONE_TIMER)

	jobId, err := jobConfig.StartJob(jobs.RunOzoneFunc, time.Minute)
	if err != nil {
		respondWithError(writer, http.StatusInternalServerError, "could not start the ozone timer")
		return
	}

	response := struct {
		JobID string `json:"job_id"`
	}{
		JobID: jobId.String(),
	}

	respondWithJSON(writer, http.StatusCreated, response)
}

func (config *serverConfig) handlerStopOzone(writer http.ResponseWriter, req *http.Request) {
	log.Println("handlerStopOzone")
	jobConfig := jobs.NewJobConfig(config.DB, jobs.JOBTYPE_OZONE_TIMER)

	jobId := req.PathValue("JOBID")
	log.Println(jobId)
	id, err := uuid.Parse(jobId)
	if err != nil {
		log.Printf("failed to parse id (%v): %v\n", jobId, err)
		respondWithError(writer, http.StatusNotFound, "invalid job id")
		return
	}

	jobConfig.StopJob(id)

}
