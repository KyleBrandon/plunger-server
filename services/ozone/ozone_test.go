package ozone

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/KyleBrandon/plunger-server/utils"
	"github.com/google/uuid"
)

func TestOzoneGet(t *testing.T) {
	t.Run("Get ozone status expect no job running", func(t *testing.T) {
		store := mockJobStore{}
		manager := mockJobManager{}
		h := NewHandler(&manager, &store)

		store.err = errors.New("could not find any ozone job")
		rr := utils.TestRequest(t, http.MethodGet, "/v1/ozone", nil, h.handlerOzoneGet)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "cound not find any ozone job") {
			t.Errorf("expected message %s, got %s", "could not find any ozone job", rr.Body.String())
		}
	})

	t.Run("Get ozone status expect a job running", func(t *testing.T) {
		store := mockJobStore{}
		manager := mockJobManager{}
		h := NewHandler(&manager, &store)

		rr := utils.TestRequest(t, http.MethodGet, "/v1/ozone", nil, h.handlerOzoneGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "seconds_left") {
			t.Errorf("expected message %s, got %s", "seconds_left", rr.Body.String())
		}
	})

	t.Run("Fail to start ozone job", func(t *testing.T) {
		store := mockJobStore{}
		manager := mockJobManager{}
		h := NewHandler(&manager, &store)

		manager.err = errors.New("could not start the ozone timer")
		rr := utils.TestRequest(t, http.MethodPost, "/v1/ozone/start", nil, h.handlerOzoneStart)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), manager.err.Error()) {
			t.Errorf("expected message %s, got %s", manager.err.Error(), rr.Body.String())
		}
	})

	t.Run("Succeed to start ozone job", func(t *testing.T) {
		store := mockJobStore{}
		manager := mockJobManager{}
		h := NewHandler(&manager, &store)

		rr := utils.TestRequest(t, http.MethodPost, "/v1/ozone/start", nil, h.handlerOzoneStart)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status code %d, got %d", http.StatusCreated, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "seconds_left") {
			t.Errorf("expected message %s, got %s", "seconds_left", rr.Body.String())
		}
	})

	t.Run("Fail to stop ozone job", func(t *testing.T) {
		store := mockJobStore{}
		manager := mockJobManager{}
		h := NewHandler(&manager, &store)

		manager.err = errors.New("could not stop the ozone job")
		rr := utils.TestRequest(t, http.MethodPost, "/v1/ozone/stop", nil, h.handlerOzoneStop)

		if rr.Code != http.StatusNotModified {
			t.Errorf("expected status code %d, got %d", http.StatusNotModified, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), manager.err.Error()) {
			t.Errorf("expected message %s, got %s", manager.err.Error(), rr.Body.String())
		}
	})

	t.Run("Succeed to stop ozone job", func(t *testing.T) {
		store := mockJobStore{}
		manager := mockJobManager{}
		h := NewHandler(&manager, &store)

		rr := utils.TestRequest(t, http.MethodPost, "/v1/ozone/stop", nil, h.handlerOzoneStop)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status code %d, got %d", http.StatusNoContent, rr.Code)
		}
	})
}

type mockJobStore struct {
	canceled bool
	err      error
	job      database.Job
	event    database.Event
}

func (m *mockJobStore) CreateJob(ctx context.Context, arg database.CreateJobParams) (database.Job, error) {
	return m.job, m.err
}

func (m *mockJobStore) GetCancelRequested(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.canceled, m.err
}

func (m *mockJobStore) GetJobById(ctx context.Context, id uuid.UUID) (database.Job, error) {
	return m.job, m.err
}

func (m *mockJobStore) GetRunningJobsByType(ctx context.Context, jobType int32) ([]database.Job, error) {
	return []database.Job{m.job}, m.err
}

func (m *mockJobStore) UpdateCancelRequested(ctx context.Context, arg database.UpdateCancelRequestedParams) (database.Job, error) {
	return m.job, m.err
}

func (m *mockJobStore) UpdateJob(ctx context.Context, arg database.UpdateJobParams) (database.Job, error) {
	return m.job, m.err
}

func (m *mockJobStore) CreateEvent(ctx context.Context, arg database.CreateEventParams) (database.Event, error) {
	return m.event, m.err
}

func (m *mockJobStore) GetLatestJobByType(ctx context.Context, jobType int32) (database.Job, error) {
	return m.job, m.err
}

type mockJobManager struct {
	err      error
	canceled bool
	job      database.Job
}

func (m *mockJobManager) StartJobWithTimeout(execute jobs.JobFunc, jobType int32, timeoutPeriod time.Duration) (*database.Job, error) {
	return &m.job, m.err
}

func (m *mockJobManager) GetRunningJob(jobType int32) (*database.Job, error) {
	return &m.job, m.err
}

func (m *mockJobManager) StartJob(execute jobs.JobFunc, jobType int32) (*database.Job, error) {
	return &m.job, m.err
}

func (m *mockJobManager) CancelJob(jobType int32) error {
	return m.err
}

func (m *mockJobManager) StopJob(jobType int32, result string) error {
	return m.err
}

func (m *mockJobManager) IsJobCanceled(jobId uuid.UUID) bool {
	return m.canceled
}
