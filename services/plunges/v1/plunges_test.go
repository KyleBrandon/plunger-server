package plunges

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/utils"
	"github.com/google/uuid"
)

func TestPlungesGet(t *testing.T) {
	t.Run("get plunge by invalid id", func(t *testing.T) {
		store := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&store, &sensors)
		rr := utils.TestRequest(t, http.MethodGet, "/v1/plunges/1234", nil, handler.handlePlungesGet)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "could not find plunge") {
			t.Errorf("received error %s", rr.Body.String())
		}
	})

	t.Run("get plunge by id that does not exist", func(t *testing.T) {
		store := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&store, &sensors)
		store.plungeID = uuid.New()
		rr := utils.TestRequest(t, http.MethodGet, "/v1/plunges/1234", nil, handler.handlePlungesGet)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "could not find plunge") {
			t.Errorf("received error %s", rr.Body.String())
		}
	})

	t.Run("get pluge that does exists", func(t *testing.T) {
		store := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&store, &sensors)
		store.plungeID = uuid.New()
		rr := utils.TestRequest(t, http.MethodGet, fmt.Sprintf("/v1/plunges/%s", store.plungeID.String()), nil, handler.handlePlungesGet)
		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}
	})
}

func TestPlungesGetAll(t *testing.T) {
	t.Run("get all plunges with error", func(t *testing.T) {
		store := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&store, &sensors)
		store.err = errors.New("failed database query")
		rr := utils.TestRequest(t, http.MethodGet, "/v1/plunges", nil, handler.handlePlungesGet)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "could not find any plunge") {
			t.Errorf("received error %s", rr.Body.String())
		}
	})

	t.Run("get all plunges", func(t *testing.T) {
		store := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&store, &sensors)
		store.plunge.ID = uuid.New()
		rr := utils.TestRequest(t, http.MethodGet, "/v1/plunges", nil, handler.handlePlungesGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), store.plunge.ID.String()) {
			t.Errorf("received error %s", rr.Body.String())
		}
	})

	t.Run("get current plunge", func(t *testing.T) {
		store := mockPlungeStore{}
		sensors := mockSensors{}

		handler := NewHandler(&store, &sensors)
		store.plunge.ID = uuid.New()

		rr := utils.TestRequest(t, http.MethodGet, "/v1/plunges?filter=current", nil, handler.handlePlungesGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), store.plunge.ID.String()) {
			t.Errorf("received error %s", rr.Body.String())
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
