package main

import (
	"log"
	"net/http"
)

func (config *serverConfig) handlerGetTemperatures(w http.ResponseWriter, r *http.Request) {
	log.Println("enter handlerGetTemperatureByLocation")

	results, err := config.Sensors.ReadTemperatures()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to read temperature sensor")
		return
	}

	respondWithJSON(w, http.StatusOK, results)

}
