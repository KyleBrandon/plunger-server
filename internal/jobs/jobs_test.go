package jobs

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/google/uuid"
)

type jobStore struct {
	Job database.Job
}

func (store *jobStore) CreateJob(ctx context.Context, arg database.CreateJobParams) (database.Job, error) {
	store.Job = database.Job{
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

func (store *jobStore) GetRunningJobsByType(ctx context.Context, jobType int32) ([]database.Job, error) {
	if store.Job.JobType == jobType {
		return []database.Job{store.Job}, nil
	}

	return nil, errors.New("job not found")
}

func (store *jobStore) GetLatestJobByType(ctx context.Context, jobType int32) (database.Job, error) {
	if store.Job.JobType == jobType {
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

func (store *jobStore) CreateEvent(ctx context.Context, arg database.CreateEventParams) (database.Event, error) {
	return database.Event{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		EventType: arg.EventType,
		EventData: arg.EventData,
	}, nil
}

func TestCreateJobTimesOut(t *testing.T) {
	jobStore := jobStore{
		Job: database.Job{},
	}

	sc := sensor.SensorConfig{}

	jobConfig := NewJobConfig(&jobStore, &sc)
	job, err := jobConfig.StartJobWithTimeout(testTimedJob, JOBTYPE_OZONE_TIMER, 500*time.Millisecond)
	if err != nil {
		t.Errorf("failed to start job: %v\n", err)
		return
	}

	// wait for job to complete
	time.Sleep(1 * time.Second)

	dbJob, _ := jobStore.GetJobById(context.Background(), job.ID)
	if !dbJob.Result.Valid || dbJob.Result.String != "Success" {
		t.Errorf("job result should be valid and 'Success', found: (%v, %v)\n", dbJob.Result.Valid, dbJob.Result.String)
		return
	}
}

func TestCreateJobWithCancel(t *testing.T) {
	jobStore := jobStore{
		Job: database.Job{},
	}

	sc := sensor.SensorConfig{}

	jobConfig := NewJobConfig(&jobStore, &sc)
	job, err := jobConfig.StartJob(testTimedJob, JOBTYPE_OZONE_TIMER)
	if err != nil {
		t.Errorf("failed to start job: %v\n", err)
		return
	}

	time.Sleep(500 * time.Millisecond)

	jobConfig.CancelJob(JOBTYPE_OZONE_TIMER)

	time.Sleep(500 * time.Millisecond)

	dbJob, _ := jobStore.GetJobById(context.Background(), job.ID)
	if !dbJob.Result.Valid || dbJob.Result.String != "Canceled" {
		t.Errorf("job result should be valid and 'Canceled', found: (%v, %v)\n", dbJob.Result.Valid, dbJob.Result.String)
		return
	}
}

func testTimedJob(config *JobConfig, ctx context.Context, cancel context.CancelFunc, jobId uuid.UUID) {
	defer cancel()

	// sensor.TurnOzoneOn()
	log.Println("Start Timed Job")
	canceled := false

	for {
		select {
		case <-ctx.Done():

			if canceled {
				config.StopJob(JOBTYPE_OZONE_TIMER, "Canceled")
			} else {
				config.StopJob(JOBTYPE_OZONE_TIMER, "Success")
			}
			return

		case <-time.After(100 * time.Millisecond):
			cancelRequested := config.IsJobCanceled(jobId)
			if cancelRequested {
				canceled = true
				cancel()
				continue
			}

		}
	}
}
