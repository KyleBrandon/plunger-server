package jobs

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/google/uuid"
)

type jobStore struct {
	Job database.Job
}

func (store *jobStore) CreateJob(ctx context.Context, arg database.CreateJobParams) (database.Job, error) {
	store.Job = database.Job{
		ID:              arg.ID,
		CreatedAt:       arg.CreatedAt,
		UpdatedAt:       arg.UpdatedAt,
		JobType:         arg.JobType,
		Status:          arg.Status,
		StartTime:       arg.StartTime,
		EndTime:         arg.EndTime,
		Result:          arg.Result,
		CancelRequested: arg.CancelRequested,
	}

	return store.Job, nil
}

func (store *jobStore) GetCancelRequested(ctx context.Context, id uuid.UUID) (bool, error) {
	return store.Job.CancelRequested, nil

}
func (store *jobStore) GetJobById(ctx context.Context, id uuid.UUID) (database.Job, error) {
	if store.Job.ID == id {
		return store.Job, nil
	}
	return database.Job{}, errors.New("job not found")
}

func (store *jobStore) UpdateCancelRequested(ctx context.Context, arg database.UpdateCancelRequestedParams) (database.Job, error) {
	if store.Job.ID == arg.ID {
		store.Job.CancelRequested = arg.CancelRequested
		return store.Job, nil
	}
	return database.Job{}, errors.New("job not found")
}

func (store *jobStore) UpdateJob(ctx context.Context, arg database.UpdateJobParams) (database.Job, error) {
	if store.Job.ID == arg.ID {
		store.Job.Status = arg.Status
		store.Job.EndTime = arg.EndTime
		store.Job.Result = arg.Result
		store.Job.CancelRequested = arg.CancelRequested

		return store.Job, nil
	}
	return database.Job{}, errors.New("job not found")
}

func TestCreateJobTimesOut(t *testing.T) {
	jobStore := jobStore{
		Job: database.Job{},
	}

	jobConfig := NewJobConfig(&jobStore, JOBTYPE_OZONE_TIMER)
	jobId, err := jobConfig.StartJob(testTimedJob, 500*time.Millisecond)
	if err != nil {
		t.Errorf("failed to start job: %v\n", err)
		return
	}

	// wait for job to complete
	time.Sleep(1 * time.Second)

	job, _ := jobStore.GetJobById(context.Background(), jobId)
	if !job.Result.Valid || job.Result.String != "Success" {
		t.Errorf("job result should be valid and 'Success', found: (%v, %v)\n", job.Result.Valid, job.Result.String)
		return
	}
}

func TestCreateJobWithCancel(t *testing.T) {
	jobStore := jobStore{
		Job: database.Job{},
	}

	jobConfig := NewJobConfig(&jobStore, JOBTYPE_OZONE_TIMER)
	jobId, err := jobConfig.StartJob(testTimedJob, 5*time.Second)
	if err != nil {
		t.Errorf("failed to start job: %v\n", err)
		return
	}

	time.Sleep(500 * time.Millisecond)

	jobConfig.StopJob(jobId)

	time.Sleep(500 * time.Millisecond)

	job, _ := jobStore.GetJobById(context.Background(), jobId)
	if !job.Result.Valid || job.Result.String != "Canceled" {
		t.Errorf("job result should be valid and 'Canceled', found: (%v, %v)\n", job.Result.Valid, job.Result.String)
		return
	}
}

func testTimedJob(config *JobConfig, ctx context.Context, cancel context.CancelFunc, jobId uuid.UUID) {
	defer cancel()

	// sensor.TurnOzoneOn()
	log.Println("Start Timed Job")
	var canceled bool = false

	for {
		select {
		case <-ctx.Done():

			resultString := "Success"
			if canceled {
				resultString = "Canceled"
			}

			// task was canceled or timed out
			result := sql.NullString{
				String: resultString,
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

		case <-time.After(500 * time.Millisecond):
			cancelRequested := config.IsJobCanceled(jobId)
			if cancelRequested {
				canceled = true
				cancel()
				continue
			}

		}
	}

}
