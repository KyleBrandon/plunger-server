package main

import (
	"log/slog"
	"net/http"
)

func (config *serverConfig) handlerPumpGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerPumpGet")

	pumpOn, err := config.Sensors.IsPumpOn()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not start the ozone timer", err)
		return
	}

	response := struct {
		PumpOn bool `json:"pump_on"`
	}{
		PumpOn: pumpOn,
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (config *serverConfig) handlerPumpStart(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerPumpStart")
	err := config.Sensors.TurnPumpOn()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to turn on the pump", err)
		return
	}

	respondWithNoContent(w, http.StatusNoContent)
}

func (config *serverConfig) handlerPumpStop(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerPumpStop")

	err := config.Sensors.TurnPumpOff()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to turn off the pump", err)
		return
	}

	respondWithNoContent(w, http.StatusNoContent)
}
