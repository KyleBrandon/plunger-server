package jobs

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/google/uuid"
)

const (
	JOBTYPE_OZONE_TIMER = 1
)

const (
	JOBSTATUS_STARTED  = 1
	JOBSTATUS_STOPPED  = 2
	JOBSTATUS_ORPHANED = 3
)

// Create an interface for the storage methods used to manage job state.  This helps for testing.
type JobStore interface {
	CreateJob(ctx context.Context, arg database.CreateJobParams) (database.Job, error)
	GetCancelRequested(ctx context.Context, id uuid.UUID) (bool, error)
	GetJobById(ctx context.Context, id uuid.UUID) (database.Job, error)
	GetRunningJobsByType(ctx context.Context, jobType int32) ([]database.Job, error)
	UpdateCancelRequested(ctx context.Context, arg database.UpdateCancelRequestedParams) (database.Job, error)
	UpdateJob(ctx context.Context, arg database.UpdateJobParams) (database.Job, error)
}

// Function signature used to define a job to run
type JobFunc func(*JobConfig, context.Context, context.CancelFunc, uuid.UUID)

// Configuration used to manage the job runner state
type JobConfig struct {
	mux *sync.Mutex
	DB  JobStore
}

func NewJobConfig(DB JobStore) *JobConfig {

	return &JobConfig{
		DB:  DB,
		mux: &sync.Mutex{},
	}
}

var ErrJobNotFound = errors.New("job was not found")

func (config *JobConfig) GetRunningJob(jobType int32) (*database.Job, error) {

	ctx := context.Background()

	ozoneJobs, err := config.DB.GetRunningJobsByType(ctx, jobType)
	if err != nil {
		return nil, err
	}

	if len(ozoneJobs) == 0 {
		log.Printf("no ozone jobs are currently running")
		return nil, ErrJobNotFound
	}

	if len(ozoneJobs) > 1 {
		log.Printf("there should only be one running ozone job, found: %v\n", len(ozoneJobs))
	}

	return &ozoneJobs[0], nil
}

func (config *JobConfig) StartJob(execute JobFunc, jobType int32, timeoutPeriod time.Duration) (*database.Job, error) {

	ctx, cancel := context.WithCancel(context.Background())
	ctx, cancel = context.WithTimeout(ctx, timeoutPeriod)

	config.mux.Lock()
	defer config.mux.Unlock()

	config.ensureOnlyOneJob(context.Background(), jobType)

	jobId := uuid.New()
	params := database.CreateJobParams{
		ID:        jobId,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		JobType:   jobType,
		Status:    JOBSTATUS_STARTED,
		StartTime: time.Now().UTC(),
	}

	params.EndTime = params.StartTime.Add(timeoutPeriod)

	job, err := config.DB.CreateJob(ctx, params)
	if err != nil {
		cancel()
		return nil, err
	}

	go execute(config, ctx, cancel, jobId)

	return &job, nil
}

func (config *JobConfig) CancelJob(jobType int32) error {
	ctx := context.Background()

	job, err := config.GetRunningJob(jobType)
	if err != nil {
		return err
	}

	params := database.UpdateCancelRequestedParams{
		ID:              job.ID,
		CancelRequested: true,
	}

	_, err = config.DB.UpdateCancelRequested(ctx, params)
	if err != nil {
		return err
	}

	return nil
}

func (config *JobConfig) StopJob(jobType int32, result string) error {
	ctx := context.Background()
	job, err := config.GetRunningJob(jobType)
	if err != nil {
		return err
	}

	sqlString := sql.NullString{
		String: result,
		Valid:  true,
	}

	// Update the database to indicate the job was canceled/stopped
	params := database.UpdateJobParams{
		ID:      job.ID,
		Status:  JOBSTATUS_STOPPED,
		EndTime: time.Now().UTC(),
		Result:  sqlString,
	}
	config.DB.UpdateJob(ctx, params)

	return nil
}

func (config *JobConfig) IsJobCanceled(jobId uuid.UUID) bool {
	log.Println("IsJobCanceled")
	ctx := context.Background()
	job, err := config.DB.GetJobById(ctx, jobId)
	if err != nil {
		log.Printf("failed to find the job %v: %v\n", jobId, err)
		return false
	}

	return job.CancelRequested
}

func (config *JobConfig) ensureOnlyOneJob(context context.Context, jobType int32) {
	currentJobs, err := config.DB.GetRunningJobsByType(context, jobType)
	if err != nil {
		log.Printf("failed to find any existing running jobs of type %v: %v\n", jobType, err)
		return
	}

	if len(currentJobs) > 0 {
		log.Printf("Found %v existing jobs of type %v\n", len(currentJobs), jobType)
		result := sql.NullString{
			String: "Canceled",
			Valid:  true,
		}
		for _, j := range currentJobs {
			jp := database.UpdateJobParams{
				ID:      j.ID,
				Status:  JOBSTATUS_ORPHANED,
				EndTime: time.Now().UTC(),
				Result:  result,
			}
			config.DB.UpdateJob(context, jp)
		}
	}
}
