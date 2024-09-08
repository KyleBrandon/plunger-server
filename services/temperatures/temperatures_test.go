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
		store := mockSensor{}
		store.err = errors.New("sensor offline")

		h := NewHandler(&store)

		rr := utils.TestRequest(t, http.MethodGet, "/v1/temperatures", nil, h.handlerTemperaturesGet)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "failed to read temperature sensor") {
			t.Errorf("received error %s, expected %s", rr.Body.String(), "failed to read temperature sensor")
		}
	})

	t.Run("should read temperature sensors", func(t *testing.T) {
		store := mockSensor{}
		h := NewHandler(&store)

		rr := utils.TestRequest(t, http.MethodGet, "/v1/temperatures", nil, h.handlerTemperaturesGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}
	})
}

type mockSensor struct {
	err          error
	temperatures []sensor.TemperatureReading
}

func (m *mockSensor) ReadTemperatures() ([]sensor.TemperatureReading, error) {
	return m.temperatures, m.err
}
