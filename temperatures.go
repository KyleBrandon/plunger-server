package main

import (
	"log/slog"
	"net/http"
)

func (config *serverConfig) handlerTemperaturesGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerTemperaturesGet")

	results, err := config.Sensors.ReadTemperatures()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to read temperature sensor", err)
		return
	}

	respondWithJSON(w, http.StatusOK, results)

}
