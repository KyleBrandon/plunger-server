package plunges

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/utils"
	"github.com/google/uuid"
)

func TestPlungesGet(t *testing.T) {
	t.Run("get plunge that has not been started", func(t *testing.T) {
		plungeStore := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&plungeStore, &sensors)
		plungeStore.plunge = database.Plunge{}
		rr := utils.TestRequest(t, http.MethodGet, "/v2/plunges/status", nil, handler.handlePlungesGet)
		utils.TestExpectedStatus(t, rr, http.StatusOK)
		// TODO: check for a valid PlungeResponse
		// utils.TestExpectedMessage(t, rr, "No active timer")
	})

	t.Run("get pluge that is running", func(t *testing.T) {
		plungeStore := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&plungeStore, &sensors)

		plungeStore.plungeID = uuid.New()
		plungeStore.plunge.Running = true
		rr := utils.TestRequest(t, http.MethodGet, "/v2/plunges/status", nil, handler.handlePlungesGet)
		utils.TestExpectedStatus(t, rr, http.StatusOK)
	})
}

func TestPlungeStart(t *testing.T) {
	t.Run("start plunge with invalid query parameter should fail", func(t *testing.T) {
		plungeStore := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&plungeStore, &sensors)

		rr := utils.TestRequest(t, http.MethodPost, "/v2/plunges/start?duration=abcd", nil, handler.handlePlungesStart)
		utils.TestExpectedStatus(t, rr, http.StatusBadRequest)
		utils.TestExpectedMessage(t, rr, "Invalid 'duration' parameter")
	})

	t.Run("start plunge without query parameter expect default 3 minute plunge", func(t *testing.T) {
		plungeStore := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&plungeStore, &sensors)

		rr := utils.TestRequest(t, http.MethodPost, "/v2/plunges/start", nil, handler.handlePlungesStart)
		utils.TestExpectedStatus(t, rr, http.StatusCreated)

		var resp PlungeResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		if err != nil {
			t.Errorf("failed to unmarshal the plunge start response: %v", err)
		}

		if resp.ExpectedDuration != 180 {
			t.Errorf("expected duration %v, got %v", 180, resp.ExpectedDuration)
		}
	})
	t.Run("start plunge with query parameter expect 4 minute plunge", func(t *testing.T) {
		plungeStore := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&plungeStore, &sensors)

		rr := utils.TestRequest(t, http.MethodPost, "/v2/plunges/start?duration=240", nil, handler.handlePlungesStart)
		utils.TestExpectedStatus(t, rr, http.StatusCreated)

		var resp PlungeResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		if err != nil {
			t.Errorf("failed to unmarshal the plunge start response: %v", err)
		}

		if resp.ExpectedDuration != 240 {
			t.Errorf("expected duration %v, got %v", 240, resp.ExpectedDuration)
		}
	})
}

type mockPlungeStore struct {
	plungeID    uuid.UUID
	plunge      database.Plunge
	temperature database.Temperature
	err         error
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
	m.plunge.Running = true
	m.plunge.StartTime.Valid = arg.StartTime.Valid
	m.plunge.StartTime.Time = arg.StartTime.Time
	m.plunge.StartWaterTemp = arg.StartWaterTemp
	m.plunge.StartRoomTemp = arg.StartRoomTemp
	m.plunge.ExpectedDuration = arg.ExpectedDuration

	return m.plunge, m.err
}

func (m *mockPlungeStore) StopPlunge(ctx context.Context, arg database.StopPlungeParams) (database.Plunge, error) {
	m.plunge.Running = false
	m.plunge.EndTime.Valid = arg.EndTime.Valid
	m.plunge.EndTime.Time = arg.EndTime.Time
	m.plunge.EndWaterTemp = arg.EndWaterTemp
	m.plunge.EndRoomTemp = arg.EndRoomTemp
	return m.plunge, m.err
}

func (m *mockPlungeStore) UpdatePlungeAvgTemp(ctx context.Context, arg database.UpdatePlungeAvgTempParams) (database.Plunge, error) {
	return m.plunge, m.err
}

func (m *mockPlungeStore) FindMostRecentTemperatures(ctx context.Context) (database.Temperature, error) {
	return m.temperature, m.err
}

func (m *mockPlungeStore) SaveTemperature(ctx context.Context, arg database.SaveTemperatureParams) (database.Temperature, error) {
	return m.temperature, m.err
}

type mockSensors struct {
	temperatures []sensor.TemperatureReading
	err          error
}

func (m *mockSensors) ReadTemperatures() []sensor.TemperatureReading {
	return m.temperatures
}

func (m *mockSensors) ReadRoomAndWaterTemperature() (sensor.TemperatureReading, sensor.TemperatureReading) {
	return sensor.TemperatureReading{}, sensor.TemperatureReading{}
}

func (m *mockSensors) IsLeakPresent() (bool, error) {
	return false, nil
}

func (m *mockSensors) TurnOzoneOn() error {
	return nil
}

func (m *mockSensors) TurnOzoneOff() error {
	return nil
}

func (m *mockSensors) IsPumpOn() (bool, error) {
	return true, nil
}

func (m *mockSensors) TurnPumpOn() error {
	return nil
}

func (m *mockSensors) TurnPumpOff() error {
	return nil
}
