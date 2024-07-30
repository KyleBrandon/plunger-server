package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/google/uuid"
)

const (
	JOBTYPE_OZONE_TIMER = 1
)

const (
	JOBSTATUS_STARTED = 1
	JOBSTATUS_STOPPED = 2
)

type JobStore interface {
	CreateJob(ctx context.Context, arg database.CreateJobParams) (database.Job, error)
	GetCancelRequested(ctx context.Context, id uuid.UUID) (bool, error)
	GetJobById(ctx context.Context, id uuid.UUID) (database.Job, error)
	UpdateCancelRequested(ctx context.Context, arg database.UpdateCancelRequestedParams) (database.Job, error)
	UpdateJob(ctx context.Context, arg database.UpdateJobParams) (database.Job, error)
}

type JobFunc func(*JobConfig, context.Context, context.CancelFunc, uuid.UUID)

type JobConfig struct {
	JobType int32
	DB      JobStore
}

func NewJobConfig(DB JobStore, jobType int32) *JobConfig {

	return &JobConfig{
		JobType: jobType,
		DB:      DB,
	}
}

func (config *JobConfig) StartJob(execute JobFunc, timeoutPeriod time.Duration) (uuid.UUID, error) {
	jobId := uuid.New()

	ctx, cancel := context.WithCancel(context.Background())
	ctx, cancel = context.WithTimeout(ctx, timeoutPeriod)

	params := database.CreateJobParams{
		ID:        jobId,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		JobType:   config.JobType,
		Status:    JOBSTATUS_STARTED,
		StartTime: time.Now(),
	}

	_, err := config.DB.CreateJob(ctx, params)
	if err != nil {
		cancel()
		return uuid.Nil, err
	}

	go execute(config, ctx, cancel, jobId)

	return jobId, nil
}

func (config *JobConfig) StopJob(jobId uuid.UUID) error {
	ctx := context.Background()
	params := database.UpdateCancelRequestedParams{
		ID:              jobId,
		CancelRequested: true,
	}

	_, err := config.DB.UpdateCancelRequested(ctx, params)
	if err != nil {
		return err
	}

	return nil
}

func RunOzoneFunc(config *JobConfig, ctx context.Context, cancel context.CancelFunc, jobId uuid.UUID) {
	defer cancel()

	sensor.TurnOzoneOn()

	for {
		select {

		case <-ctx.Done():
			// task was canceled or timedout
			sensor.TurnOzoneOff()

			result := sql.NullString{
				String: "Success",
				Valid:  true,
			}

			// Update the database to indicate the job was canceled/stopped
			params := database.UpdateJobParams{
				ID:      jobId,
				Status:  JOBSTATUS_STOPPED,
				EndTime: time.Now(),
				Result:  result,
			}
			config.DB.UpdateJob(ctx, params)
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

func (config *JobConfig) IsJobCanceled(jobId uuid.UUID) bool {
	log.Println("IsJobCanceled")
	ctx := context.Background()
	job, err := config.DB.GetJobById(ctx, jobId)
	if err != nil {
		log.Println("failed to query the job")

		return false
	}

	return job.CancelRequested
}
