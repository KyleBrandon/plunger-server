package temperatures

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/utils"
)

func TestReadTemperatures(t *testing.T) {
	t.Run("should fail to read if sensors are non-responsive", func(t *testing.T) {
		store := mockSensors{}
		store.temperatures = []sensor.TemperatureReading{
			{
				Err: errors.New("failed to read temperature"),
			},
		}

		h := NewHandler(&store)

		rr := utils.TestRequest(t, http.MethodGet, "/v1/temperatures", nil, h.handlerTemperaturesGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "failed to read temperature") {
			t.Errorf("received error %s, expected %s", rr.Body.String(), "failed to read temperature")
		}
	})

	t.Run("should read temperature sensors", func(t *testing.T) {
		store := mockSensors{}
		h := NewHandler(&store)

		rr := utils.TestRequest(t, http.MethodGet, "/v1/temperatures", nil, h.handlerTemperaturesGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}
	})
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
