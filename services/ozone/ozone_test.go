package ozone

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/utils"
	"github.com/google/uuid"
)

func TestOzoneGet(t *testing.T) {
	t.Run("Get ozone status expect no job running", func(t *testing.T) {
		store := mockOzoneStore{}
		sensors := mockSensors{}
		h := NewHandler(&store, &sensors)

		store.SetError(errors.New("could not find any ozone job"))
		rr := utils.TestRequest(t, http.MethodGet, "/v1/ozone", nil, h.handlerOzoneGet)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "cound not find any ozone job") {
			t.Errorf("expected message %s, got %s", "could not find any ozone job", rr.Body.String())
		}
	})

	t.Run("Get ozone status expect a job running", func(t *testing.T) {
		store := mockOzoneStore{}
		sensors := mockSensors{}
		h := NewHandler(&store, &sensors)

		rr := utils.TestRequest(t, http.MethodGet, "/v1/ozone", nil, h.handlerOzoneGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "expected_duration") {
			t.Errorf("expected message %s, got %s", "expected_duration", rr.Body.String())
		}
	})

	t.Run("Fail to start ozone job", func(t *testing.T) {
		store := mockOzoneStore{}
		sensors := mockSensors{}
		h := NewHandler(&store, &sensors)

		store.SetError(errors.New("could not start the ozone timer"))
		rr := utils.TestRequest(t, http.MethodPost, "/v1/ozone/start", nil, h.handlerOzoneStart)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
		}

		e := *store.err
		if !strings.Contains(rr.Body.String(), e.Error()) {
			t.Errorf("expected message %s, got %s", e.Error(), rr.Body.String())
		}
	})

	t.Run("Succeed to start ozone job", func(t *testing.T) {
		store := mockOzoneStore{}
		sensors := mockSensors{}
		h := NewHandler(&store, &sensors)

		rr := utils.TestRequest(t, http.MethodPost, "/v1/ozone/start", nil, h.handlerOzoneStart)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status code %d, got %d", http.StatusCreated, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "\"running\":true") {
			t.Errorf("expected message %s, got %s", "\"running\":true", rr.Body.String())
		}
	})

	t.Run("Fail to stop ozone job", func(t *testing.T) {
		store := mockOzoneStore{}
		sensors := mockSensors{}
		h := NewHandler(&store, &sensors)

		store.SetError(errors.New("could not find ozone job"))
		rr := utils.TestRequest(t, http.MethodPost, "/v1/ozone/stop", nil, h.handlerOzoneStop)

		if rr.Code != http.StatusNotModified {
			t.Errorf("expected status code %d, got %d", http.StatusNotModified, rr.Code)
		}

		e := *store.err
		if !strings.Contains(rr.Body.String(), e.Error()) {
			t.Errorf("expected message %s, got %s", e.Error(), rr.Body.String())
		}
	})

	t.Run("Succeed to stop ozone job", func(t *testing.T) {
		store := mockOzoneStore{}
		sensors := mockSensors{}
		store.entry.Running = true
		h := NewHandler(&store, &sensors)

		rr := utils.TestRequest(t, http.MethodPost, "/v1/ozone/stop", nil, h.handlerOzoneStop)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status code %d, got %d", http.StatusNoContent, rr.Code)
		}
	})
}

type mockOzoneStore struct {
	entry database.Ozone
	err   *error
}

func (m *mockOzoneStore) SetError(err error) {
	m.err = &err
}

func (m *mockOzoneStore) GetLatestOzoneEntry(ctx context.Context) (database.Ozone, error) {
	if m.err != nil {
		return database.Ozone{}, *m.err
	}

	return m.entry, nil
}

func (m *mockOzoneStore) StartOzoneGenerator(ctx context.Context, arg database.StartOzoneGeneratorParams) (database.Ozone, error) {
	if m.err != nil {
		return database.Ozone{}, *m.err
	}

	m.entry.Running = true

	return m.entry, nil
}

func (m *mockOzoneStore) StopOzoneGenerator(ctx context.Context, id uuid.UUID) (database.Ozone, error) {
	if m.err != nil {
		return database.Ozone{}, *m.err
	}

	return m.entry, nil
}

func (m *mockOzoneStore) UpdateOzoneEntryStatus(ctx context.Context, arg database.UpdateOzoneEntryStatusParams) (database.Ozone, error) {
	if m.err != nil {
		return database.Ozone{}, *m.err
	}

	return m.entry, nil
}

type mockSensors struct {
	temperatures []sensor.TemperatureReading
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
