package leaks

import (
	"context"
	"encoding/json"
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

func TestStartMonitoring(t *testing.T) {
	t.Run("Fail to start monitoring for leaks", func(t *testing.T) {
		store := mockLeakStore{}
		manager := mockJobManager{}

		h := NewHandler(&manager, &store)

		manager.err = errors.New("failed to start monitoring for leaks")
		err := h.StartMonitoringLeaks()
		if err == nil {
			t.Error("Expected start monitoring for leaks to fail")
		}
	})

	t.Run("Succeed to start monitoring for leaks", func(t *testing.T) {
		store := mockLeakStore{}
		manager := mockJobManager{}

		h := NewHandler(&manager, &store)

		err := h.StartMonitoringLeaks()
		if err != nil {
			t.Error("Expected start monitoring for leaks to succeed")
		}
	})
}

func TestGetCurrentLeak(t *testing.T) {
	t.Run("Fail to find a currently running leak", func(t *testing.T) {
		store := mockLeakStore{}
		manager := mockJobManager{}

		h := NewHandler(&manager, &store)

		store.err = errors.New("could not find the current leak event")
		rr := utils.TestRequest(t, http.MethodGet, "/v1/leaks?filter=current", nil, h.handlerLeakGet)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), store.err.Error()) {
			t.Errorf("expected message %s, got %s", store.err.Error(), rr.Body.String())
		}
	})

	t.Run("Invalid current leak data", func(t *testing.T) {
		store := mockLeakStore{}
		manager := mockJobManager{}

		h := NewHandler(&manager, &store)
		rr := utils.TestRequest(t, http.MethodGet, "j/v1/leaks?filter=current", nil, h.handlerLeakGet)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "invalid leaks") {
			t.Errorf("expected message %s, got %s", "invalid leaks", rr.Body.String())
		}
	})

	t.Run("Succeed in finding currently running leak", func(t *testing.T) {
		store := mockLeakStore{}
		manager := mockJobManager{}

		leakData := DbLeakEvent{
			EventTime:     time.Now().UTC(),
			PreviousState: false,
			CurrentState:  true,
		}

		eventData, err := json.Marshal(leakData)
		if err != nil {
			t.Error("failed to marshal leak test data")
		}

		store.leak.EventData = eventData
		store.leak.EventType = EVENTTYPE_LEAK
		h := NewHandler(&manager, &store)
		rr := utils.TestRequest(t, http.MethodGet, "/v1/leaks?filter=current", nil, h.handlerLeakGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "leak_detected") {
			t.Errorf("expected message %s, got %s", "leak_detected", rr.Body.String())
		}
	})
}

func TestGetAllLeaks(t *testing.T) {
	t.Run("Fail to query all leaks", func(t *testing.T) {
		store := mockLeakStore{}
		manager := mockJobManager{}

		store.err = errors.New("failed to read all leaks")

		h := NewHandler(&manager, &store)
		rr := utils.TestRequest(t, http.MethodGet, "/v1/leaks", nil, h.handlerLeakGet)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), store.err.Error()) {
			t.Errorf("expected message %s, got %s", store.err.Error(), rr.Body.String())
		}
	})

	t.Run("Invalid leak data", func(t *testing.T) {
		store := mockLeakStore{}
		manager := mockJobManager{}

		h := NewHandler(&manager, &store)
		rr := utils.TestRequest(t, http.MethodGet, "/v1/leaks", nil, h.handlerLeakGet)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "invalid leaks") {
			t.Errorf("expected message %s, got %s", "invalid leaks", rr.Body.String())
		}
	})

	t.Run("Succeed in finding all leaks", func(t *testing.T) {
		store := mockLeakStore{}
		manager := mockJobManager{}

		leakData := DbLeakEvent{
			EventTime:     time.Now().UTC(),
			PreviousState: false,
			CurrentState:  true,
		}

		eventData, err := json.Marshal(leakData)
		if err != nil {
			t.Error("failed to marshal leak test data")
		}
		store.leak.EventData = eventData
		store.leak.EventType = EVENTTYPE_LEAK

		h := NewHandler(&manager, &store)
		rr := utils.TestRequest(t, http.MethodGet, "/v1/leaks", nil, h.handlerLeakGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "leak_detected") {
			t.Errorf("expected message %s, got %s", "", rr.Body.String())
		}
	})
}

type mockLeakStore struct {
	err  error
	leak database.Event
}

func (m *mockLeakStore) GetLatestEventByType(ctx context.Context, eventType int32) (database.Event, error) {
	return m.leak, m.err
}

func (m *mockLeakStore) GetEventsByType(ctx context.Context, arg database.GetEventsByTypeParams) ([]database.Event, error) {
	return []database.Event{m.leak}, m.err
}

type mockJobManager struct {
	err      error
	job      database.Job
	canceled bool
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
