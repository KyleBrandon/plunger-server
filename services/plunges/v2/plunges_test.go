package plunges

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/utils"
	"github.com/google/uuid"
)

func TestPlungesGet(t *testing.T) {
	t.Run("get plunge that has not been started", func(t *testing.T) {
		store := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&store, &sensors)
		store.plungeID = uuid.New()
		rr := utils.TestRequest(t, http.MethodGet, "/v2/plunges/status", nil, handler.handlePlungesGet)
		utils.TestExpectedStatus(t, rr, http.StatusNotFound)
		utils.TestExpectedMessage(t, rr, "No active timer")
	})

	t.Run("get pluge that is running", func(t *testing.T) {
		store := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&store, &sensors)
		handler.Running = true

		store.plungeID = uuid.New()
		rr := utils.TestRequest(t, http.MethodGet, "/v2/plunges/status", nil, handler.handlePlungesGet)
		utils.TestExpectedStatus(t, rr, http.StatusOK)
	})
}

func TestPlungeStart(t *testing.T) {
	t.Run("start plunge with invalid query parameter should fail", func(t *testing.T) {
		store := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&store, &sensors)
		rr := utils.TestRequest(t, http.MethodPost, "/v2/plunges/start?duration=abcd", nil, handler.handlePlungesStart)
		utils.TestExpectedStatus(t, rr, http.StatusBadRequest)
		utils.TestExpectedMessage(t, rr, "Invalid 'duration' parameter")
	})

	t.Run("start plunge without query parameter expect default 3 minute plunge", func(t *testing.T) {
		store := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&store, &sensors)
		rr := utils.TestRequest(t, http.MethodPost, "/v2/plunges/start", nil, handler.handlePlungesStart)
		utils.TestExpectedStatus(t, rr, http.StatusCreated)

		d := time.Duration(3) * time.Minute
		if handler.Duration != d {
			t.Errorf("expected duration %v, got %v", d, handler.Duration)
		}
	})
	t.Run("start plunge with query parameter expect 4 minute plunge", func(t *testing.T) {
		store := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&store, &sensors)
		rr := utils.TestRequest(t, http.MethodPost, "/v2/plunges/start?duration=240", nil, handler.handlePlungesStart)
		utils.TestExpectedStatus(t, rr, http.StatusCreated)

		d := time.Duration(4) * time.Minute
		if handler.Duration != d {
			t.Errorf("expected duration %v, got %v", d, handler.Duration)
		}
	})
}

type mockPlungeStore struct {
	plungeID uuid.UUID
	plunge   database.Plunge
	err      error
}

func (m *mockPlungeStore) GetLatestPlunge(ctx context.Context) (database.Plunge, error) {
	return m.plunge, m.err
}

func (m *mockPlungeStore) GetPlungeByID(ctx context.Context, id uuid.UUID) (database.Plunge, error) {
	if id != m.plungeID {
		return m.plunge, errors.New("invalid plunge id")
	}
	return m.plunge, m.err
}

func (m *mockPlungeStore) GetPlunges(ctx context.Context) ([]database.Plunge, error) {
	result := []database.Plunge{m.plunge}
	return result, m.err
}

func (m *mockPlungeStore) StartPlunge(ctx context.Context, arg database.StartPlungeParams) (database.Plunge, error) {
	return m.plunge, m.err
}

func (m *mockPlungeStore) StopPlunge(ctx context.Context, arg database.StopPlungeParams) (database.Plunge, error) {
	return m.plunge, m.err
}

func (m *mockPlungeStore) UpdatePlungeStatus(ctx context.Context, arg database.UpdatePlungeStatusParams) (database.Plunge, error) {
	return m.plunge, m.err
}

type mockSensors struct {
	temperatures []sensor.TemperatureReading
	err          error
}

func (m *mockSensors) ReadTemperatures() ([]sensor.TemperatureReading, error) {
	return m.temperatures, m.err
}
