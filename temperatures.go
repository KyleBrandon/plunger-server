package main

import (
	"log"
	"net/http"
)

func (config *serverConfig) handlerTemperaturesGet(w http.ResponseWriter, r *http.Request) {
	log.Println("handlerTemperaturesGet")

	results, err := config.Sensors.ReadTemperatures()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to read temperature sensor")
		return
	}

	respondWithJSON(w, http.StatusOK, results)

}
