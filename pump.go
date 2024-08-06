package main

import (
	"log"
	"net/http"
)

func (config *serverConfig) handlerPumpGet(w http.ResponseWriter, r *http.Request) {
	log.Println("handlerPumpGet")

	pumpOn, err := config.Sensors.IsPumpOn()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not start the ozone timer")
		return
	}

	response := struct {
		PumpOn bool `json:"pump_on"`
	}{
		PumpOn: pumpOn,
	}

	respondWithJSON(w, http.StatusOK, response)
}
