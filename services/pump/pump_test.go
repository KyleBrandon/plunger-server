package pump

import (
	"errors"
	"net/http"
	"testing"

	"github.com/KyleBrandon/plunger-server/utils"
)

func TestPumpStatusIsOn(t *testing.T) {
	pumpSensor := mockPumpSensor{
		on:  true,
		err: nil,
	}
	handler := NewHandler(&pumpSensor)

	t.Run("should return pump is on", func(t *testing.T) {
		pumpSensor.err = nil

		rr := utils.TestRequest(t, http.MethodGet, "/v1/pump", nil, handler.handlerPumpGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("should fail pump is on", func(t *testing.T) {
		pumpSensor.err = errors.New("failed to start pump")

		rr := utils.TestRequest(t, http.MethodGet, "/v1/pump", nil, handler.handlerPumpGet)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})
}

func TestPumpPower(t *testing.T) {
	pumpSensor := mockPumpSensor{
		on:  true,
		err: nil,
	}
	handler := NewHandler(&pumpSensor)

	t.Run("should turn pump on", func(t *testing.T) {
		pumpSensor.err = nil

		rr := utils.TestRequest(t, http.MethodPost, "/v1/pump", nil, handler.handlerPumpStart)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status code %d, got %d", http.StatusNoContent, rr.Code)
		}
	})

	t.Run("should fail to turn pump on", func(t *testing.T) {
		pumpSensor.err = errors.New("failed")

		rr := utils.TestRequest(t, http.MethodPost, "/v1/pump", nil, handler.handlerPumpStart)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("should turn pump off", func(t *testing.T) {
		pumpSensor.err = nil

		rr := utils.TestRequest(t, http.MethodPost, "/v1/pump", nil, handler.handlerPumpStop)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status code %d, got %d", http.StatusNoContent, rr.Code)
		}
	})

	t.Run("should fail to turn pump off", func(t *testing.T) {
		pumpSensor.err = errors.New("failed")

		rr := utils.TestRequest(t, http.MethodPost, "/v1/pump", nil, handler.handlerPumpStop)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})
}

type mockPumpSensor struct {
	on  bool
	err error
}

func (m *mockPumpSensor) IsPumpOn() (bool, error) {
	return m.on, m.err
}

func (m *mockPumpSensor) TurnPumpOn() error {
	m.on = true
	return m.err
}

func (m *mockPumpSensor) TurnPumpOff() error {
	m.on = false
	return m.err
}
