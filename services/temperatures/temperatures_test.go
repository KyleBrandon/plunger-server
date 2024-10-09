package temperatures

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/utils"
)

func TestReadTemperatures(t *testing.T) {
	t.Run("should fail to read if sensors are non-responsive", func(t *testing.T) {
		store := mockStore{}
		s := mockSensors{}
		s.temperatures = []sensor.TemperatureReading{
			{
				Err: errors.New("failed to read temperature"),
			},
		}

		h := NewHandler(&store, &s)

		rr := utils.TestRequest(t, http.MethodGet, "/v1/temperatures", nil, h.handlerTemperaturesGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "failed to read temperature") {
			t.Errorf("received error %s, expected %s", rr.Body.String(), "failed to read temperature")
		}
	})

	t.Run("should read temperature sensors", func(t *testing.T) {
		store := mockStore{}
		sensor := mockSensors{}
		h := NewHandler(&store, &sensor)

		rr := utils.TestRequest(t, http.MethodGet, "/v1/temperatures", nil, h.handlerTemperaturesGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}
	})
}

type mockStore struct {
	temperature database.Temperature
	err         error
}

func (m *mockStore) FindMostRecentTemperatures(ctx context.Context) (database.Temperature, error) {
	return m.temperature, m.err
}

func (m *mockStore) SaveTemperature(ctx context.Context, arg database.SaveTemperatureParams) (database.Temperature, error) {
	return m.temperature, m.err
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
